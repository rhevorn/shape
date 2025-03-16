package shape_test

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/types"
)

type contractProfile struct {
	Bio string `json:"bio"`
}

type contractMetadata struct{ Source string }

type contractUser struct {
	Name     string
	Age      int
	Count    int64
	Ratio    float64
	Code     uint16
	Enabled  bool
	Created  time.Time
	Timeout  types.Duration
	Nickname *string
	Tags     []string
	Scores   map[string]int
	Profile  contractProfile
	Metadata contractMetadata
}

// compilePublicAPI is intentionally not executed. Compiling this package locks
// the documented root API names, type relationships, and method signatures.
func compilePublicAPI(ctx context.Context, reader io.Reader, source []byte) {
	identityString := func(value string) (string, error) { return value, nil }
	identityStringContext := func(context.Context, string) (string, error) { return "", nil }
	checkString := func(string) error { return nil }
	checkStringContext := func(context.Context, string) error { return nil }

	stringSpec := shape.String().
		IfZero("default").
		Trim().Trim("_").
		LTrim().LTrim("_").
		RTrim().RTrim("_").
		ToLower().ToUpper().
		Apply(identityString).
		ApplyContext(identityStringContext).
		NotEmpty().MinLength(1).MaxLength(10).Len(3).
		OneOf("one", "two").Pattern(`^x`).
		StartsWith("x").EndsWith("x").Contains("x").
		Email().URL().UUID().IP().
		Refine(checkString).
		RefineContext(checkStringContext).
		Label("string")
	_ = stringSpec.Pointer()
	_ = stringSpec.Slice()
	_, _ = stringSpec.Transform("")
	_, _ = stringSpec.TransformContext(ctx, "")
	_ = stringSpec.Validate("")
	_ = stringSpec.ValidateContext(ctx, "")
	_ = stringSpec.ValidateFirst("")
	_ = stringSpec.ValidateFirstContext(ctx, "")
	var _ shape.Schema[string] = stringSpec

	numberSpec := shape.Number[uint16]().
		IfZero(1).
		Apply(func(value uint16) (uint16, error) { return value, nil }).
		ApplyContext(func(context.Context, uint16) (uint16, error) { return 0, nil }).
		Min(1).Max(10).Gt(0).Gte(1).Lt(11).Lte(10).
		Between(1, 10).OneOf(1, 2).
		Positive().Negative().NonNegative().
		Refine(func(uint16) error { return nil }).
		RefineContext(func(context.Context, uint16) error { return nil }).
		Label("number")
	_ = numberSpec.Pointer()
	_ = numberSpec.Slice()
	_, _ = numberSpec.Transform(1)
	_ = numberSpec.Validate(1)
	var _ shape.Schema[uint16] = numberSpec

	valueSpec := shape.Value[contractMetadata]().
		IfZero(contractMetadata{}).
		Apply(func(value contractMetadata) (contractMetadata, error) { return value, nil }).
		ApplyContext(func(context.Context, contractMetadata) (contractMetadata, error) { return contractMetadata{}, nil }).
		Refine(func(contractMetadata) error { return nil }).
		RefineContext(func(context.Context, contractMetadata) error { return nil }).
		Label("value")
	_ = valueSpec.Pointer()
	_ = valueSpec.Slice()
	_, _ = valueSpec.Transform(contractMetadata{})
	_ = valueSpec.Validate(contractMetadata{})
	var _ shape.Schema[contractMetadata] = valueSpec

	fallback := "fallback"
	pointerSpec := shape.Pointer("Nickname", shape.String()).
		IfNull(&fallback).
		Apply(func(value *string) (*string, error) { return value, nil }).
		ApplyContext(func(context.Context, *string) (*string, error) { return nil, nil }).
		NotNull().NotEmpty().
		Refine(func(*string) error { return nil }).
		RefineContext(func(context.Context, *string) error { return nil }).
		Label("pointer")
	_, _ = pointerSpec.Transform(nil)
	_ = pointerSpec.Validate(nil)
	var _ shape.Schema[*string] = pointerSpec

	sliceSpec := shape.Slice("Tags", shape.String()).
		IfNull([]string{}).
		Apply(func(value []string) ([]string, error) { return value, nil }).
		ApplyContext(func(context.Context, []string) ([]string, error) { return nil, nil }).
		NotNull().NotEmpty().Min(1).Max(10).Len(2).Unique().
		Refine(func([]string) error { return nil }).
		RefineContext(func(context.Context, []string) error { return nil }).
		Label("slice")
	_, _ = sliceSpec.Transform(nil)
	_ = sliceSpec.Validate(nil)
	var _ shape.Schema[[]string] = sliceSpec

	mapSpec := shape.Map("Scores", shape.String(), shape.Int()).
		IfNull(map[string]int{}).
		Apply(func(value map[string]int) (map[string]int, error) { return value, nil }).
		ApplyContext(func(context.Context, map[string]int) (map[string]int, error) { return nil, nil }).
		NotNull().NotEmpty().Min(1).Max(10).Len(2).
		Refine(func(map[string]int) error { return nil }).
		RefineContext(func(context.Context, map[string]int) error { return nil }).
		Label("map")
	_, _ = mapSpec.Transform(nil)
	_ = mapSpec.Validate(nil)
	var _ shape.Schema[map[string]int] = mapSpec

	profileSchema := shape.New[contractProfile](shape.String("Bio"))
	schema := shape.New[contractUser](
		shape.String("Name"),
		shape.Int("Age"),
		shape.Int64("Count"),
		shape.Float64("Ratio"),
		shape.Number[uint16]("Code"),
		shape.Bool("Enabled"),
		shape.Time("Created"),
		shape.Duration("Timeout"),
		pointerSpec,
		sliceSpec,
		mapSpec,
		shape.Field("Profile", profileSchema),
		shape.Value[contractMetadata]("Metadata"),
	).Apply(
		func(value contractUser) (contractUser, error) { return value, nil },
	).ApplyContext(
		func(context.Context, contractUser) (contractUser, error) { return contractUser{}, nil },
	).Refine(
		func(contractUser) error { return errors.New("compile only") },
	).RefineContext(
		func(context.Context, contractUser) error { return nil },
	)

	var schemaInterface shape.Schema[contractUser] = schema
	_, _ = schemaInterface.Transform(contractUser{})
	_, _ = schemaInterface.TransformContext(ctx, contractUser{})
	_ = schemaInterface.Validate(contractUser{})
	_ = schemaInterface.ValidateContext(ctx, contractUser{})
	_ = schemaInterface.ValidateFirst(contractUser{})
	_ = schemaInterface.ValidateFirstContext(ctx, contractUser{})
	_, _ = schemaInterface.ParseJSON(source)
	_, _ = schemaInterface.ParseJSONContext(ctx, source)
	_, _ = schemaInterface.ParseJSONReader(reader)
	_, _ = schemaInterface.ParseJSONReaderContext(ctx, reader)

	var target contractUser
	_ = schemaInterface.BindJSON(&target, source)
	_ = schemaInterface.BindJSONContext(ctx, &target, source)
	_ = schemaInterface.BindJSONReader(&target, reader)
	_ = schemaInterface.BindJSONReaderContext(ctx, &target, reader)

	_ = shape.Struct[contractUser]()
	_ = shape.BindJSON(&target, source)
	_ = shape.BindJSONContext(ctx, &target, source)
	_ = shape.BindJSONReader(&target, reader)
	_ = shape.BindJSONReaderContext(ctx, &target, reader)
}

var _ = compilePublicAPI
