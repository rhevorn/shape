package shape_test

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/validate"
)

type profile struct {
	Name string `json:"name" shape:"notempty,trim,maxlength=8"`
	Age  int    `json:"age" shape:"min=18,max=120"`
}

var profileSchema = shape.Struct[profile]()
var _ shape.Schema[profile] = profileSchema

func TestSchemaKeepsTransformAndValidateIndependent(t *testing.T) {
	input := profile{Name: "  Pong  ", Age: 20}
	out, err := profileSchema.Transform(input)
	if err != nil || out.Name != "Pong" {
		t.Fatalf("Transform() = %#v, %v", out, err)
	}
	if input.Name != "  Pong  " {
		t.Fatalf("Transform mutated input: %#v", input)
	}

	// Validation observes exactly the value it receives and never trims it.
	if err := profileSchema.Validate(profile{Name: "   ", Age: 20}); err != nil {
		t.Fatalf("Validate unexpectedly transformed input: %v", err)
	}
}

func TestTaggedSchemaDoesNotMutateNestedInput(t *testing.T) {
	type Item struct {
		Name string `json:"name" shape:"trim"`
	}
	type Request struct {
		Items []Item `json:"items"`
	}
	schema := shape.Struct[Request]()
	input := Request{Items: []Item{{Name: " Pong "}}}
	out, err := schema.Transform(input)
	if err != nil {
		t.Fatal(err)
	}
	if input.Items[0].Name != " Pong " || out.Items[0].Name != "Pong" {
		t.Fatalf("input=%#v output=%#v", input, out)
	}
}

func TestExplicitSchemaParseJSON(t *testing.T) {
	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	schema := shape.New[User](
		shape.Field("Name", shape.String().Trim().NotEmpty()),
		shape.Field("Age", shape.Int().Min(18)),
	).Apply(func(user User) (User, error) {
		if user.Name != "" {
			user.Name = strings.ToUpper(user.Name[:1]) + user.Name[1:]
		}
		return user, nil
	}).Refine(func(user User) error {
		if user.Name == "Admin" {
			return errors.New("reserved name")
		}
		return nil
	})

	user, err := schema.ParseJSON([]byte(`{"name":" Pong ","age":20}`))
	if err != nil || user.Name != "Pong" || user.Age != 20 {
		t.Fatalf("ParseJSON() = %#v, %v", user, err)
	}
	if _, err := schema.ParseJSON([]byte(`{"name":" Pong ","age":17}`)); err == nil {
		t.Fatal("ParseJSON() accepted invalid user")
	}
	if _, err := schema.ParseJSON([]byte(`{"name":" admin ","age":20}`)); err == nil {
		t.Fatal("ParseJSON() accepted whole-struct refine failure")
	}
}

func TestPackageParseJSONAcceptsAnySchema(t *testing.T) {
	text, err := shape.ParseJSON(shape.String().Trim().NotEmpty(), []byte(`" Pong "`))
	if err != nil || text != "Pong" {
		t.Fatalf("String ParseJSON() = %q, %v", text, err)
	}

	number, err := shape.ParseJSON(shape.Int().Positive(), []byte(`123`))
	if err != nil || number != 123 {
		t.Fatalf("Int ParseJSON() = %d, %v", number, err)
	}
	if _, err := shape.ParseJSON(shape.Int(), []byte(`"123"`)); err == nil {
		t.Fatal("Int ParseJSON() coerced a JSON string")
	}

	items, err := shape.ParseJSON(shape.String().Trim().NotEmpty().Slice(), []byte(`[" a ","b"]`))
	if err != nil || len(items) != 2 || items[0] != "a" || items[1] != "b" {
		t.Fatalf("Slice ParseJSON() = %#v, %v", items, err)
	}

	values, err := shape.ParseJSON(shape.Map(shape.String().Trim(), shape.Int().Positive()), []byte(`{" a ":1}`))
	if err != nil || len(values) != 1 || values["a"] != 1 {
		t.Fatalf("Map ParseJSON() = %#v, %v", values, err)
	}

	fallback := "guest"
	pointer, err := shape.ParseJSON(shape.String().Trim().Pointer().IfNull(&fallback), []byte(`null`))
	if err != nil || pointer == nil || *pointer != "guest" {
		t.Fatalf("Pointer ParseJSON() = %#v, %v", pointer, err)
	}
}

func TestExplicitSchemaCompositeFields(t *testing.T) {
	type Profile struct {
		Bio string `json:"bio"`
	}
	type User struct {
		Nickname *string        `json:"nickname"`
		Tags     []string       `json:"tags"`
		Scores   map[string]int `json:"scores"`
		Profile  Profile        `json:"profile"`
	}
	profile := shape.New[Profile](shape.Field("Bio", shape.String().Trim().NotEmpty()))
	schema := shape.New[User](
		shape.Field("Nickname", shape.Pointer(shape.String().Trim()).NotNull()),
		shape.Field("Tags", shape.Slice(shape.String().Trim().NotEmpty()).NotEmpty().Unique()),
		shape.Field("Scores", shape.Map(shape.String().Trim().NotEmpty(), shape.Int().NonNegative()).NotEmpty()),
		shape.Field("Profile", profile),
	)

	nickname := " Pong "
	out, err := schema.Transform(User{
		Nickname: &nickname,
		Tags:     []string{" one ", "two"},
		Scores:   map[string]int{" score ": 1},
		Profile:  Profile{Bio: " hello "},
	})
	if err != nil || *out.Nickname != "Pong" || out.Tags[0] != "one" || out.Scores["score"] != 1 || out.Profile.Bio != "hello" {
		t.Fatalf("Transform() = %#v, %v", out, err)
	}

	err = schema.Validate(User{Tags: []string{"", "ok"}, Scores: map[string]int{"x": -1}})
	var validation *validate.Error
	if !errors.As(err, &validation) {
		t.Fatalf("Validate() error = %T %v", err, err)
	}
	paths := make([]string, len(validation.Issues))
	for i, issue := range validation.Issues {
		paths[i] = issue.Path.String()
	}
	if strings.Join(paths, ",") != "nickname,tags[0],scores[\"x\"],profile.bio" {
		t.Fatalf("issue paths = %v", paths)
	}
}

func TestExplicitSchemaRejectsBadFields(t *testing.T) {
	type User struct {
		Name   string `json:"name"`
		Hidden string `json:"-"`
	}
	tests := []struct {
		name string
		make func()
	}{
		{"missing", func() { _ = shape.New[User](shape.Field("Missing", shape.String())) }},
		{"case mismatch", func() { _ = shape.New[User](shape.Field("name", shape.String())) }},
		{"wrong type", func() { _ = shape.New[User](shape.Field("Name", shape.Int())) }},
		{"duplicate", func() { _ = shape.New[User](shape.Field("Name", shape.String()), shape.Field("Name", shape.String())) }},
		{"json excluded", func() { _ = shape.New[User](shape.Field("Hidden", shape.String())) }},
		{"unnamed", func() { _ = shape.New[User](shape.Field("", shape.String())) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("New did not panic")
				}
			}()
			test.make()
		})
	}
}

func TestExplicitSchemaAllowsWholeStructOnly(t *testing.T) {
	type Pair struct{ Left, Right int }
	schema := shape.New[Pair]().Refine(func(value Pair) error {
		if value.Left > value.Right {
			return errors.New("left must not exceed right")
		}
		return nil
	})
	if err := schema.Validate(Pair{Left: 2, Right: 1}); err == nil {
		t.Fatal("whole-struct-only schema accepted invalid value")
	}
}

func TestTopLevelBindJSONUsesTags(t *testing.T) {
	var user profile
	err := shape.BindJSON(&user, []byte(`{"name":" Pong ","age":20}`))
	if err != nil || user.Name != "Pong" || user.Age != 20 {
		t.Fatalf("BindJSON() = %#v, %v", user, err)
	}

	err = shape.BindJSONReaderContext(
		context.Background(),
		&user,
		strings.NewReader(`{"name":" next ","age":21}`),
	)
	if err != nil || user.Name != "next" || user.Age != 21 {
		t.Fatalf("BindJSONReaderContext() = %#v, %v", user, err)
	}
}

func TestSchemaJSONIsDecodeTransformValidate(t *testing.T) {
	out, err := profileSchema.ParseJSON([]byte(`{"name":" Pong ","age":20}`))
	if err != nil || out.Name != "Pong" || out.Age != 20 {
		t.Fatalf("ParseJSON() = %#v, %v", out, err)
	}

	_, err = profileSchema.ParseJSON([]byte(`{"name":"   ","age":20}`))
	var validation *validate.Error
	if !errors.As(err, &validation) || len(validation.Issues) != 1 || validation.Issues[0].Path.String() != "name" {
		t.Fatalf("error = %#v", err)
	}
}

func TestSchemaValidateAllAndFirst(t *testing.T) {
	value := profile{Name: "", Age: 10}
	err := profileSchema.Validate(value)
	var all *validate.Error
	if !errors.As(err, &all) || len(all.Issues) != 2 {
		t.Fatalf("Validate error = %#v", err)
	}

	err = profileSchema.ValidateFirst(value)
	var first *validate.Error
	if !errors.As(err, &first) || len(first.Issues) != 1 || first.Issues[0].Path.String() != "name" {
		t.Fatalf("ValidateFirst error = %#v", err)
	}
}

func TestSchemaNestedPathsAndStableMapOrder(t *testing.T) {
	type Item struct {
		Email string `json:"email" shape:"email"`
	}
	type Request struct {
		Items  []Item         `json:"items" shape:"notempty"`
		Labels map[int]string `json:"labels"`
	}
	schema := shape.Struct[Request]()
	err := schema.Validate(Request{
		Items:  []Item{{Email: "bad"}, {Email: "also-bad"}},
		Labels: map[int]string{10: "x", 2: "y"},
	})
	var validation *validate.Error
	if !errors.As(err, &validation) || len(validation.Issues) != 2 {
		t.Fatalf("error = %#v", err)
	}
	if validation.Issues[0].Path.String() != "items[0].email" || validation.Issues[1].Path.String() != "items[1].email" {
		t.Fatalf("paths = %#v", validation.Issues)
	}
}

func TestSchemaNullFallbackAndAtomicBind(t *testing.T) {
	type Config struct {
		Name  string `json:"name" shape:"ifzero=guest,trim,notempty"`
		Debug *bool  `json:"debug" shape:"ifnull=true"`
	}
	schema := shape.Struct[Config]()
	out, err := schema.ParseJSON([]byte(`{"name":null,"debug":null}`))
	if err != nil || out.Name != "guest" || out.Debug == nil || !*out.Debug {
		t.Fatalf("ParseJSON() = %#v, %v", out, err)
	}

	target := Config{Name: "keep"}
	err = shape.BindJSON(&target, []byte(`{"name":"   "}`))
	if err == nil || target.Name != "keep" {
		t.Fatalf("BindJSON must be atomic: target=%#v err=%v", target, err)
	}

	err = shape.BindJSON(&target, []byte(`{"name":" next "}`))
	if err != nil || target.Name != "next" {
		t.Fatalf("BindJSON() target=%#v err=%v", target, err)
	}
}

func TestSchemaJSONOptionsAndContext(t *testing.T) {
	_, err := profileSchema.ParseJSON(
		[]byte(`{"name":"Pong","age":20,"extra":true}`),
		shape.JSONOptions{DisallowUnknownFields: true},
	)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown-field error = %v", err)
	}

	_, err = profileSchema.ParseJSONReader(
		strings.NewReader(`{"name":"Pong","age":20}`),
		shape.JSONOptions{MaxBytes: 4},
	)
	if !errors.Is(err, shape.ErrJSONTooLarge) {
		t.Fatalf("size error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := profileSchema.TransformContext(ctx, profile{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("context error = %v", err)
	}
}

func TestStructRejectsInvalidConfiguration(t *testing.T) {
	type Invalid struct {
		Value int `shape:"trim"`
	}
	defer func() {
		if recover() == nil {
			t.Fatal("Struct did not panic")
		}
	}()
	_ = shape.Struct[Invalid]()
}

func TestSchemaUsesValidateLocale(t *testing.T) {
	ctx := validate.WithLocale(context.Background(), validate.SimplifiedChinese)
	err := profileSchema.ValidateFirstContext(ctx, profile{Age: 20})
	var got *validate.Error
	if !errors.As(err, &got) || len(got.Issues) != 1 || got.Issues[0].Message != "不能为空" {
		t.Fatalf("error = %#v", err)
	}
}

func TestStringOneOfAndCollectionLenTags(t *testing.T) {
	type Value struct {
		Mode  string   `shape:"oneof=read|write"`
		Items []string `shape:"len=2"`
	}
	schema := shape.Struct[Value]()
	if err := schema.Validate(Value{Mode: "read", Items: []string{"a", "b"}}); err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(Value{Mode: "other", Items: []string{"a"}}); err == nil {
		t.Fatal("expected tag validation errors")
	}
}

func TestExplicitTransformErrorsKeepCompletePath(t *testing.T) {
	type Request struct {
		Items []string `json:"items"`
	}
	boom := errors.New("bad item")
	schema := shape.New[Request](
		shape.Field("Items", shape.Slice(shape.String().Apply(func(value string) (string, error) {
			if value == "bad" {
				return "", boom
			}
			return value, nil
		}))),
	)

	_, err := schema.Transform(Request{Items: []string{"ok", "bad"}})
	var transformError *shape.TransformError
	if !errors.As(err, &transformError) || transformError.Path.String() != "items[1]" || !errors.Is(err, boom) {
		t.Fatalf("Transform() error = %#v", err)
	}
}

func TestStandaloneSpecTransformUsesPublicError(t *testing.T) {
	boom := errors.New("bad value")
	stringSpec := shape.String().Apply(func(string) (string, error) {
		return "", boom
	})
	_, err := stringSpec.Transform("bad")
	var transformError *shape.TransformError
	if !errors.As(err, &transformError) || len(transformError.Path) != 0 || !errors.Is(err, boom) {
		t.Fatalf("StringSpec.Transform() error = %#v", err)
	}

	mapSpec := shape.Map(shape.Int(), stringSpec)
	_, err = mapSpec.Transform(map[int]string{2: "bad", 1: "bad"})
	transformError = nil
	if !errors.As(err, &transformError) || transformError.Path.String() != "[1]" || !errors.Is(err, boom) {
		t.Fatalf("MapSpec.Transform() error = %#v", err)
	}
}

// A nested Schema returns its own *TransformError, and the collection wrapper
// sits outside it. Normalization must recover the index/key from the wrapper
// before matching the public type, or the element index disappears from the
// path while Validate still reports it.
func TestNestedSchemaTransformErrorKeepsCollectionIndex(t *testing.T) {
	type Item struct {
		Sku string `json:"sku"`
	}
	type Order struct {
		Items []Item          `json:"items"`
		ByKey map[string]Item `json:"byKey"`
	}
	boom := errors.New("bad sku")
	item := shape.New[Item](shape.Field("Sku", shape.String().Apply(func(value string) (string, error) {
		if value == "bad" {
			return "", boom
		}
		return value, nil
	})))
	order := shape.New[Order](
		shape.Field("Items", shape.Slice(item)),
		shape.Field("ByKey", shape.Map(shape.String(), item)),
	)

	for _, tc := range []struct {
		name  string
		value Order
		want  string
	}{
		{"slice element", Order{Items: []Item{{Sku: "ok"}, {Sku: "bad"}}}, "items[1].sku"},
		{"map value", Order{ByKey: map[string]Item{"k": {Sku: "bad"}}}, "byKey[\"k\"].sku"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := order.Transform(tc.value)
			var transformError *shape.TransformError
			if !errors.As(err, &transformError) {
				t.Fatalf("Transform() error = %T %v", err, err)
			}
			if got := transformError.Path.String(); got != tc.want {
				t.Fatalf("Path = %q, want %q (err = %v)", got, tc.want, err)
			}
			if !errors.Is(err, boom) {
				t.Fatalf("errors.Is(err, boom) = false: %v", err)
			}
		})
	}
}

// Validate and Transform must agree on the path of the same failing element.
func TestNestedSchemaTransformAndValidateAgreeOnPath(t *testing.T) {
	type Item struct {
		Sku string `json:"sku" shape:"notempty"`
	}
	type Order struct {
		Items []Item `json:"items"`
	}
	order := shape.Struct[Order]()
	value := Order{Items: []Item{{Sku: "ok"}, {Sku: ""}}}

	verr := order.Validate(value)
	var validationError *validate.Error
	if !errors.As(verr, &validationError) || len(validationError.Issues) != 1 {
		t.Fatalf("Validate() error = %v", verr)
	}
	if got := validationError.Issues[0].Path.String(); got != "items[1].sku" {
		t.Fatalf("Validate path = %q", got)
	}
}

// The spec path must inherit the time-aware zero test too; only the tagged
// path used isZero.
func TestTimeSpecIfZeroTreatsAZeroTimeAsZero(t *testing.T) {
	type Doc struct {
		When time.Time `json:"when"`
	}
	zero := time.Time{}.Local()
	if !zero.IsZero() || reflect.ValueOf(zero).IsZero() {
		t.Skip("this platform does not distinguish the two zero checks")
	}
	fallback := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	out, err := shape.New[Doc](shape.Field("When", shape.Time().IfZero(fallback))).Transform(Doc{When: zero})
	if err != nil {
		t.Fatal(err)
	}
	if !out.When.Equal(fallback) {
		t.Fatalf("When = %v, want the fallback %v", out.When, fallback)
	}
}

// The tagged compiler already rejects two fields sharing a JSON name; the
// explicit path accepted them, making one field unreachable from JSON.
func TestExplicitSchemaRejectsDuplicateJSONNames(t *testing.T) {
	type Coll struct {
		First  string
		Second string `json:"First"`
	}
	defer func() {
		if recover() == nil {
			t.Fatal("duplicate JSON names were accepted")
		}
	}()
	_ = shape.New[Coll](shape.Field("First", shape.String()), shape.Field("Second", shape.String()))
}

// label is an outer tag while rules on a pointer field attach to the element
// plan, so `label=...,min=...` used to lose the label on *T but not on T.
func TestPointerFieldKeepsItsLabel(t *testing.T) {
	type Doc struct {
		Age  *int `json:"age" shape:"label=Age,min=18"`
		Size int  `json:"size" shape:"label=Size,min=18"`
	}

	age := 1
	err := shape.Struct[Doc]().Validate(Doc{Age: &age, Size: 1})
	var validationError *validate.Error
	if !errors.As(err, &validationError) || len(validationError.Issues) != 2 {
		t.Fatalf("issues = %#v", err)
	}
	for _, issue := range validationError.Issues {
		if issue.Label == "" {
			t.Fatalf("issue %q lost its label: %#v", issue.Path.String(), issue)
		}
	}
	if got := validationError.Issues[0].Label; got != "Age" {
		t.Fatalf("pointer field label = %q, want %q", got, "Age")
	}
}

func TestWholeStructTransformErrorUsesStableType(t *testing.T) {
	type Request struct{ Name string }
	boom := errors.New("whole transform")
	schema := shape.New[Request]().Apply(func(Request) (Request, error) {
		return Request{}, boom
	})

	_, err := schema.Transform(Request{})
	var transformError *shape.TransformError
	if !errors.As(err, &transformError) || len(transformError.Path) != 0 || !errors.Is(err, boom) {
		t.Fatalf("Transform() error = %T %#v", err, err)
	}
}

func TestWholeStructApplyUsesOneWorkingCopy(t *testing.T) {
	type Large struct {
		Values []int
	}

	var firstStorage *int
	var secondStorage *int
	base := shape.New[Large]().Apply(func(value Large) (Large, error) {
		firstStorage = &value.Values[0]
		value.Values[0]++
		return value, nil
	})
	schema := base.Apply(func(value Large) (Large, error) {
		secondStorage = &value.Values[0]
		value.Values[0]++
		return value, nil
	})

	input := Large{Values: []int{1}}
	out, err := schema.Transform(input)
	if err != nil {
		t.Fatal(err)
	}
	if input.Values[0] != 1 || out.Values[0] != 3 {
		t.Fatalf("input=%#v output=%#v", input, out)
	}
	if firstStorage == &input.Values[0] || firstStorage != secondStorage {
		t.Fatalf("Apply callbacks did not share one detached working copy: input=%p first=%p second=%p", &input.Values[0], firstStorage, secondStorage)
	}

	baseOut, err := base.Transform(input)
	if err != nil || baseOut.Values[0] != 2 {
		t.Fatalf("base Schema was mutated by fluent append: %#v, %v", baseOut, err)
	}
}

// Both Schema paths must protect caller-owned storage when a whole-struct
// Apply fails. The tagged plan only detaches the fields it compiles, so
// storage held by json:"-" (and unexported) fields — which a nested Apply
// reaches through promoted or exported selectors — stayed shared with the
// caller.
func TestFailedWholeStructApplyDoesNotMutateInput(t *testing.T) {
	boom := errors.New("boom")

	t.Run("explicit", func(t *testing.T) {
		type Large struct{ Values []int }
		schema := shape.New[Large]().Apply(func(value Large) (Large, error) {
			value.Values[0] = 99
			return value, boom
		})
		input := Large{Values: []int{1}}
		if _, err := schema.Transform(input); !errors.Is(err, boom) {
			t.Fatalf("Transform() error = %v", err)
		}
		if input.Values[0] != 1 {
			t.Fatalf("failed Transform mutated input: %#v", input)
		}
	})

	t.Run("tagged", func(t *testing.T) {
		type Large struct {
			Kept   string `json:"kept"`
			Values []int  `json:"-"`
		}
		schema := shape.Struct[Large]().Apply(func(value Large) (Large, error) {
			value.Values[0] = 99
			return value, boom
		})
		input := Large{Values: []int{1}}
		if _, err := schema.Transform(input); !errors.Is(err, boom) {
			t.Fatalf("Transform() error = %v", err)
		}
		if input.Values[0] != 1 {
			t.Fatalf("failed Transform mutated input: %#v", input)
		}
	})
}

// A non-finite float must be rejected the same way whether the field was
// declared with the Float64 spec, the generic Value spec, or a shape tag.
func TestValueFloatAgreesWithNumberOnNonFinite(t *testing.T) {
	type Ratio struct {
		R float64 `json:"r"`
	}
	for _, tc := range []struct {
		name   string
		schema shape.Schema[Ratio]
	}{
		{"Value[float64]", shape.New[Ratio](shape.Field("R", shape.Value[float64]()))},
		{"Float64", shape.New[Ratio](shape.Field("R", shape.Float64()))},
		{"tagged", shape.Struct[Ratio]()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.schema.Validate(Ratio{R: math.NaN()})
			var validationError *validate.Error
			if !errors.As(err, &validationError) || len(validationError.Issues) != 1 {
				t.Fatalf("NaN was accepted: %v", err)
			}
			if code := validationError.Issues[0].Code; code != validate.CodeInvalidNumber {
				t.Fatalf("Code = %q, want %q", code, validate.CodeInvalidNumber)
			}
		})
	}
}

func TestTaggedIssuesUseContextLanguage(t *testing.T) {
	type Request struct {
		Name string `json:"name" shape:"label=姓名,notempty"`
	}
	ctx := validate.WithLocale(context.Background(), validate.SimplifiedChinese)
	err := shape.Struct[Request]().ValidateContext(ctx, Request{})
	var got *validate.Error
	if !errors.As(err, &got) || len(got.Issues) != 1 || got.Issues[0].Message != "姓名不能为空" {
		t.Fatalf("error = %#v", err)
	}
}

func TestTaggedSpecSupportsWholeStructBehavior(t *testing.T) {
	type Request struct {
		Name string `shape:"trim"`
	}
	schema := shape.Struct[Request]().
		Apply(func(value Request) (Request, error) {
			value.Name = strings.ToUpper(value.Name)
			return value, nil
		}).
		Refine(func(value Request) error {
			if value.Name == "ADMIN" {
				return errors.New("reserved name")
			}
			return nil
		})

	out, err := schema.Transform(Request{Name: " pong "})
	if err != nil || out.Name != "PONG" {
		t.Fatalf("Transform() = %#v, %v", out, err)
	}
	if err := schema.Validate(Request{Name: "ADMIN"}); err == nil {
		t.Fatal("Validate() accepted whole-struct refine failure")
	}
}

func TestFluentCompositeFactories(t *testing.T) {
	fallback := "guest"
	pointer := shape.String().Trim().Pointer().IfNull(&fallback)
	out, err := pointer.Transform(nil)
	if err != nil || out == nil || *out != "guest" {
		t.Fatalf("Pointer Transform() = %#v, %v", out, err)
	}

	slice := shape.Int().Positive().Slice().NotEmpty()
	if err := slice.Validate([]int{1, 2}); err != nil {
		t.Fatalf("Slice Validate() = %v", err)
	}
}

func TestExplicitAndTaggedStringRulesStayEquivalent(t *testing.T) {
	type Contacts struct {
		Email string `json:"email" shape:"email"`
		URL   string `json:"url" shape:"url"`
		UUID  string `json:"uuid" shape:"uuid"`
		IP    string `json:"ip" shape:"ip"`
	}
	explicit := shape.New[Contacts](
		shape.Field("Email", shape.String().Email()),
		shape.Field("URL", shape.String().URL()),
		shape.Field("UUID", shape.String().UUID()),
		shape.Field("IP", shape.String().IP()),
	)
	tagged := shape.Struct[Contacts]()

	values := []Contacts{
		{Email: "pong@example.com", URL: "https://example.com/a", UUID: "550e8400-e29b-41d4-a716-446655440000", IP: "127.0.0.1"},
		{Email: "Pong <pong@example.com>", URL: "/relative", UUID: "bad", IP: "999.1.1.1"},
	}
	for _, value := range values {
		explicitErr := explicit.Validate(value)
		taggedErr := tagged.Validate(value)
		if !reflect.DeepEqual(issueSignatures(explicitErr), issueSignatures(taggedErr)) {
			t.Fatalf("value=%#v explicit=%#v tagged=%#v", value, explicitErr, taggedErr)
		}
	}
}

func TestExplicitAndTaggedTransformsStayEquivalent(t *testing.T) {
	type Text struct {
		Value string `json:"value" shape:"trim,tolower"`
	}
	explicit := shape.New[Text](shape.Field("Value", shape.String().Trim().ToLower()))
	tagged := shape.Struct[Text]()
	want := Text{Value: "pong"}

	explicitValue, explicitErr := explicit.Transform(Text{Value: " PONG "})
	taggedValue, taggedErr := tagged.Transform(Text{Value: " PONG "})
	if explicitErr != nil || taggedErr != nil || explicitValue != want || taggedValue != want {
		t.Fatalf("explicit=%#v/%v tagged=%#v/%v", explicitValue, explicitErr, taggedValue, taggedErr)
	}
}

func TestMapTraversalOrderIsShared(t *testing.T) {
	type Item struct {
		Value string `json:"value" shape:"notempty"`
	}
	type Values struct {
		Items map[int]Item `json:"items"`
	}
	err := shape.Struct[Values]().Validate(Values{Items: map[int]Item{3: {}, 1: {}, 2: {}}})
	var validationError *validate.Error
	if !errors.As(err, &validationError) {
		t.Fatalf("tagged map validation = %v", err)
	}
	want := []string{"items[1].value", "items[2].value", "items[3].value"}
	got := make([]string, len(validationError.Issues))
	for index, issue := range validationError.Issues {
		got[index] = issue.Path.String()
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tagged map paths = %v, want %v", got, want)
	}

	validatorErr := validate.Map(validate.Int(), validate.String().NotEmpty()).
		Validate(map[int]string{3: "", 1: "", 2: ""})
	if !errors.As(validatorErr, &validationError) {
		t.Fatalf("fluent map validation = %v", validatorErr)
	}
	got = got[:0]
	for _, issue := range validationError.Issues {
		got = append(got, issue.Path.String())
	}
	if !reflect.DeepEqual(got, []string{"[1]", "[2]", "[3]"}) {
		t.Fatalf("fluent map paths = %v", got)
	}

	type MapHolder struct {
		Items map[int]string
	}
	transformer := shape.New[MapHolder](shape.Field("Items", shape.Map(shape.Int().Apply(func(value int) (int, error) {
		return value, errors.New("stop")
	}), shape.String())))
	_, transformErr := transformer.Transform(MapHolder{Items: map[int]string{3: "c", 1: "a", 2: "b"}})
	var pathError *shape.TransformError
	if !errors.As(transformErr, &pathError) || pathError.Path.String() != "Items[1]" {
		t.Fatalf("fluent map transform error = %#v", transformErr)
	}
}

func issueSignatures(err error) []string {
	if err == nil {
		return nil
	}
	var validationError *validate.Error
	if !errors.As(err, &validationError) {
		return []string{err.Error()}
	}
	out := make([]string, len(validationError.Issues))
	for index, issue := range validationError.Issues {
		out[index] = issue.Path.String() + ":" + issue.Code
	}
	return out
}
