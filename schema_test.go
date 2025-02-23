package shape_test

import (
	"context"
	"errors"
	"strings"
	"testing"

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

func TestExplicitSchemaParseJSON(t *testing.T) {
	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	schema := shape.New[User](
		shape.String("Name").Trim().NotEmpty(),
		shape.Int("Age").Min(18),
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
	profile := shape.New[Profile](shape.String("Bio").Trim().NotEmpty())
	schema := shape.New[User](
		shape.Pointer("Nickname", shape.String().Trim()).NotNull(),
		shape.Slice("Tags", shape.String().Trim().NotEmpty()).NotEmpty().Unique(),
		shape.Map("Scores", shape.String().Trim().NotEmpty(), shape.Int().NonNegative()).NotEmpty(),
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
	if strings.Join(paths, ",") != "nickname,tags[0],scores.x,profile.bio" {
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
		{"missing", func() { _ = shape.New[User](shape.String("Missing")) }},
		{"wrong type", func() { _ = shape.New[User](shape.Int("Name")) }},
		{"duplicate", func() { _ = shape.New[User](shape.String("Name"), shape.String("Name")) }},
		{"json excluded", func() { _ = shape.New[User](shape.String("Hidden")) }},
		{"unnamed", func() { _ = shape.New[User](shape.String()) }},
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
	err = schema.BindJSON(&target, []byte(`{"name":"   "}`))
	if err == nil || target.Name != "keep" {
		t.Fatalf("BindJSON must be atomic: target=%#v err=%v", target, err)
	}

	err = schema.BindJSON(&target, []byte(`{"name":" next "}`))
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
		shape.Slice("Items", shape.String().Apply(func(value string) (string, error) {
			if value == "bad" {
				return "", boom
			}
			return value, nil
		})),
	)

	_, err := schema.Transform(Request{Items: []string{"ok", "bad"}})
	var transformError *shape.TransformError
	if !errors.As(err, &transformError) || transformError.Path.String() != "items[1]" || !errors.Is(err, boom) {
		t.Fatalf("Transform() error = %#v", err)
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

func TestTaggedIssuesCanBeRelocalized(t *testing.T) {
	type Request struct {
		Name string `json:"name" shape:"label=姓名,notempty"`
	}
	err := shape.Struct[Request]().Validate(Request{})

	localized := validate.Localize(err, validate.SimplifiedChinese)
	var got *validate.Error
	if !errors.As(localized, &got) || len(got.Issues) != 1 || got.Issues[0].Message != "姓名不能为空" {
		t.Fatalf("localized error = %#v", localized)
	}
	english := validate.Localize(localized, validate.English)
	if !errors.As(english, &got) || got.Issues[0].Message != "姓名 must not be empty" {
		t.Fatalf("English error = %#v", english)
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
