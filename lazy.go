package shape

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// The non-zero-sized marker guarantees distinct pointers for independently
// constructed lazy schemas.
type lazyIdentity struct{ marker byte }

type lazyState[T any] struct {
	once     sync.Once
	provider func() Schema[T]
	schema   Schema[T]
	err      error
	id       *lazyIdentity
}

// DefaultMaxRecursiveDepth bounds recursive Lazy parsing when MaxDepth is not
// specified. It is intentionally lower than encoding/json's nesting ceiling
// so validation paths cannot amplify a compact input into excessive work.
const DefaultMaxRecursiveDepth = 64

type lazyDepthKey struct{}

type lazyDepthState struct {
	mu     sync.Mutex
	depths map[*lazyIdentity]int
}

func (s *lazyDepthState) enter(identity *lazyIdentity, limit int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	depth := s.depths[identity] + 1
	if depth > limit {
		return false
	}
	s.depths[identity] = depth
	return true
}

func (s *lazyDepthState) leave(identity *lazyIdentity) {
	s.mu.Lock()
	defer s.mu.Unlock()
	depth := s.depths[identity] - 1
	if depth == 0 {
		delete(s.depths, identity)
		return
	}
	s.depths[identity] = depth
}

// LazySchema defers construction of a named schema until its first use. The
// provider is evaluated at most once, which permits schemas to refer to
// themselves while remaining safe for concurrent reuse.
type LazySchema[T any] struct {
	name     string
	state    *lazyState[T]
	maxDepth int
}

// Lazy creates a named, lazily resolved schema. Names identify definitions
// when exporting recursive JSON Schema documents and must be unique within a
// document.
func Lazy[T any](name string, provider func() Schema[T]) LazySchema[T] {
	if name == "" {
		panic("shape: lazy schema name must not be empty")
	}
	if provider == nil {
		panic("shape: lazy schema provider must not be nil")
	}
	return LazySchema[T]{
		name:     name,
		maxDepth: DefaultMaxRecursiveDepth,
		state: &lazyState[T]{
			provider: provider,
			id:       &lazyIdentity{},
		},
	}
}

// MaxDepth sets the maximum active recursion depth for this named Lazy schema.
// The limit must be positive and is enforced independently for each Lazy
// identity in a mutually recursive graph.
func (s LazySchema[T]) MaxDepth(limit int) LazySchema[T] {
	if limit <= 0 {
		panic("shape: lazy schema maximum depth must be positive")
	}
	s.maxDepth = limit
	return s
}

func (s LazySchema[T]) resolve() (Schema[T], error) {
	if s.state == nil {
		return nil, errors.New("shape: uninitialized lazy schema")
	}
	s.state.once.Do(func() {
		s.state.err = errors.New("shape: lazy schema provider did not complete")
		s.state.schema = s.state.provider()
		if s.state.schema == nil {
			s.state.err = errors.New("shape: lazy schema provider returned nil")
			return
		}
		s.state.err = nil
		s.state.provider = nil
	})
	return s.state.schema, s.state.err
}

// Parse implements Schema[T].
func (s LazySchema[T]) Parse(value any) (T, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[T].
func (s LazySchema[T]) ParseContext(ctx context.Context, value any) (T, error) {
	var zero T
	if err := checkContext(ctx); err != nil {
		return zero, err
	}
	if s.state == nil {
		return zero, errors.New("shape: uninitialized lazy schema")
	}
	limit := s.maxDepth
	if limit == 0 {
		limit = DefaultMaxRecursiveDepth
	}
	depthState, _ := ctx.Value(lazyDepthKey{}).(*lazyDepthState)
	if depthState == nil {
		depthState = &lazyDepthState{depths: make(map[*lazyIdentity]int)}
		ctx = context.WithValue(ctx, lazyDepthKey{}, depthState)
	}
	if !depthState.enter(s.state.id, limit) {
		return zero, validationError(Issue{
			Code:     CodeTooDeep,
			Message:  "maximum recursive schema depth exceeded",
			Expected: limit,
			Received: limit + 1,
		})
	}
	defer depthState.leave(s.state.id)
	resolved, err := s.resolve()
	if err != nil {
		return zero, err
	}
	return resolved.ParseContext(ctx, value)
}

func (s LazySchema[T]) buildJSONSchema(ctx *jsonSchemaBuildContext) (map[string]any, error) {
	if s.state == nil {
		return nil, errors.New("shape: uninitialized lazy schema")
	}
	if owner, exists := ctx.owners[s.name]; exists && owner != s.state.id {
		return nil, fmt.Errorf("shape: duplicate lazy schema name %q", s.name)
	}
	ctx.owners[s.name] = s.state.id

	reference := map[string]any{"$ref": "#/$defs/" + escapeJSONPointerToken(s.name)}
	if _, exists := ctx.definitions[s.name]; exists || ctx.building[s.name] {
		return reference, nil
	}

	ctx.building[s.name] = true
	resolved, err := s.resolve()
	if err != nil {
		delete(ctx.building, s.name)
		return nil, err
	}
	definition, err := buildJSONSchemaWithContext(resolved, ctx)
	delete(ctx.building, s.name)
	if err != nil {
		return nil, err
	}
	ctx.definitions[s.name] = definition
	return reference, nil
}

func escapeJSONPointerToken(value string) string {
	value = strings.ReplaceAll(value, "~", "~0")
	return strings.ReplaceAll(value, "/", "~1")
}
