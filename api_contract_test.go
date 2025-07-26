package shape_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/types"
	"github.com/rhevorn/shape/validate"
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

// compilePublicAPI checks the documented types and signatures at compile time.
// It is not called at runtime.
func compilePublicAPI(ctx context.Context, reader io.Reader, source []byte) {
	_ = &shape.UnsupportedSchemaError{Feature: "compile only"}
	_ = validate.Path{validate.MapKeyPath(1)}
	_ = validate.PathMapKey
	_ = validate.Issue{Target: validate.TargetKey}
	var _ validate.IssueTarget = validate.TargetValue

	// Value factories take no field name; Field binds a schema to a field.
	var _ func() shape.ValueSpec[string] = shape.Value[string]
	var _ func() shape.StringSpec = shape.String
	var _ func() shape.NumberSpec[uint16] = shape.Number[uint16]
	var _ func() shape.NumberSpec[int] = shape.Int
	var _ func() shape.NumberSpec[int64] = shape.Int64
	var _ func() shape.NumberSpec[float64] = shape.Float64
	var _ func() shape.NumberSpec[types.Duration] = shape.Duration
	var _ func() shape.ValueSpec[bool] = shape.Bool
	var _ func() shape.ValueSpec[time.Time] = shape.Time
	var _ func(shape.Schema[string]) shape.PointerSpec[string] = shape.Pointer[string]
	var _ func(shape.Schema[string]) shape.SliceSpec[string] = shape.Slice[string]
	var _ func(shape.Schema[string], shape.Schema[int]) shape.MapSpec[string, int] = shape.Map[string, int]
	var _ func(string, shape.Schema[string]) shape.FieldSpec = shape.Field[string]

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
	pointerSpec := shape.Pointer(shape.String()).
		IfNull(&fallback).
		Apply(func(value *string) (*string, error) { return value, nil }).
		ApplyContext(func(context.Context, *string) (*string, error) { return nil, nil }).
		NotNull().
		Refine(func(*string) error { return nil }).
		RefineContext(func(context.Context, *string) error { return nil }).
		Label("pointer")
	_, _ = pointerSpec.Transform(nil)
	_ = pointerSpec.Validate(nil)
	var _ shape.Schema[*string] = pointerSpec

	sliceSpec := shape.Slice(shape.String()).
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

	mapSpec := shape.Map(shape.String(), shape.Int()).
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

	profileSchema := shape.New[contractProfile](shape.Field("Bio", shape.String()))
	schema := shape.New[contractUser](
		shape.Field("Name", shape.String()),
		shape.Field("Age", shape.Int()),
		shape.Field("Count", shape.Int64()),
		shape.Field("Ratio", shape.Float64()),
		shape.Field("Code", shape.Number[uint16]()),
		shape.Field("Enabled", shape.Bool()),
		shape.Field("Created", shape.Time()),
		shape.Field("Timeout", shape.Duration()),
		shape.Field("Nickname", pointerSpec),
		shape.Field("Tags", sliceSpec),
		shape.Field("Scores", mapSpec),
		shape.Field("Profile", profileSchema),
		shape.Field("Metadata", shape.Value[contractMetadata]()),
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

	var jsonSchema shape.JSONSchema[contractUser] = schema
	_, _ = jsonSchema.ParseJSON(source)
	_, _ = jsonSchema.ParseJSONContext(ctx, source)
	_, _ = jsonSchema.ParseJSONReader(reader)
	_, _ = jsonSchema.ParseJSONReaderContext(ctx, reader)

	_, _ = shape.ParseJSON(shape.String().Trim(), source)
	_, _ = shape.ParseJSONContext(ctx, shape.Int().Positive(), source)
	_, _ = shape.ParseJSONReader(shape.String().Slice(), reader)
	_, _ = shape.ParseJSONReaderContext(ctx, shape.Int().Slice(), reader)

	var target contractUser
	_ = shape.FromTags[contractUser]()
	_ = shape.BindJSON(&target, source)
	_ = shape.BindJSONContext(ctx, &target, source)
	_ = shape.BindJSONReader(&target, reader)
	_ = shape.BindJSONReaderContext(ctx, &target, reader)
}

var _ = compilePublicAPI

func compileParameterAPI(ctx context.Context, values url.Values) {
	var _ error = &shape.ParameterError{Source: "form", Path: validate.Path{validate.FieldPath("bio")}, Err: errors.New("invalid")}
	form := shape.FormOptions{DisallowUnknownFields: true}
	query := shape.QueryOptions{DisallowUnknownFields: true}
	tagged, explicit := shape.FromTags[contractProfile](), shape.New[contractProfile]()
	var target contractProfile

	var _ func(shape.Schema[contractProfile], url.Values, ...shape.FormOptions) (contractProfile, error) = shape.ParseForm[contractProfile]
	var _ func(context.Context, shape.Schema[contractProfile], url.Values, ...shape.FormOptions) (contractProfile, error) = shape.ParseFormContext[contractProfile]
	var _ func(*contractProfile, url.Values, ...shape.FormOptions) error = shape.BindForm[contractProfile]
	var _ func(context.Context, *contractProfile, url.Values, ...shape.FormOptions) error = shape.BindFormContext[contractProfile]
	_, _ = tagged.ParseForm(values, form)
	_, _ = tagged.ParseFormContext(ctx, values, form)
	_, _ = explicit.ParseForm(values, form)
	_, _ = explicit.ParseFormContext(ctx, values, form)
	_ = shape.BindForm(&target, values, form)
	_ = shape.BindFormContext(ctx, &target, values, form)

	var _ func(shape.Schema[contractProfile], url.Values, ...shape.QueryOptions) (contractProfile, error) = shape.ParseQuery[contractProfile]
	var _ func(context.Context, shape.Schema[contractProfile], url.Values, ...shape.QueryOptions) (contractProfile, error) = shape.ParseQueryContext[contractProfile]
	var _ func(*contractProfile, url.Values, ...shape.QueryOptions) error = shape.BindQuery[contractProfile]
	var _ func(context.Context, *contractProfile, url.Values, ...shape.QueryOptions) error = shape.BindQueryContext[contractProfile]
	_, _ = tagged.ParseQuery(values, query)
	_, _ = tagged.ParseQueryContext(ctx, values, query)
	_, _ = explicit.ParseQuery(values, query)
	_, _ = explicit.ParseQueryContext(ctx, values, query)
	_ = shape.BindQuery(&target, values, query)
	_ = shape.BindQueryContext(ctx, &target, values, query)
}

var _ = compileParameterAPI

func compileRequestAPI(request *http.Request) {
	var _ func(*contractProfile, *http.Request, ...shape.RequestOptions) error = shape.BindRequest[contractProfile]
	var _ error = &shape.SourceConflictError{Field: "Bio", Sources: []string{"query", "json"}}
	var _ shape.RequestPrecedence = shape.RejectConflicts
	var _ shape.RequestPrecedence = shape.QueryFirst
	var _ shape.RequestPrecedence = shape.BodyFirst
	var _ int64 = shape.DefaultMaxRequestBytes
	_ = shape.ErrRequestTooLarge
	_ = shape.ErrUnsupportedContentType
	_ = shape.ErrRequestFiles
	var target contractProfile
	_ = shape.BindRequest(&target, request, shape.RequestOptions{DisallowUnknownFields: true, MaxBytes: 1024, Precedence: shape.QueryFirst})
}

var _ = compileRequestAPI
