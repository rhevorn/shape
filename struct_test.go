package shape

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseShapeTag(t *testing.T) {
	t.Parallel()
	options, err := parseShapeTag(`trim,min=2,max=50,label='姓名',email`)
	if err != nil {
		t.Fatal(err)
	}
	want := []tagOption{
		{name: "trim"},
		{name: "min", value: "2", has: true},
		{name: "max", value: "50", has: true},
		{name: "label", value: "姓名", has: true},
		{name: "email"},
	}
	if len(options) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(options), len(want), options)
	}
	for i := range want {
		if options[i] != want[i] {
			t.Fatalf("options[%d] = %#v, want %#v", i, options[i], want[i])
		}
	}
}

func TestParseShapeTagQuotedComma(t *testing.T) {
	t.Parallel()
	options, err := parseShapeTag(`label="a,b",startswith='x,y'`)
	if err != nil {
		t.Fatal(err)
	}
	if len(options) != 2 || options[0].value != "a,b" || options[1].value != "x,y" {
		t.Fatalf("unexpected options: %#v", options)
	}
}

type createUserRequest struct {
	Name  string `json:"name" shape:"trim,min=2,max=50,label='姓名'"`
	Email string `json:"email" shape:"trim,email,label='邮箱'"`
	Age   int    `json:"age" shape:"min=18,label='年龄'"`
}

func TestMustStructCreateUser(t *testing.T) {
	t.Parallel()
	schema := MustStruct[createUserRequest]().Strict()
	user, err := Parse(schema, `{"name":" Pong ","email":"pong@example.com","age":30}`)
	if err != nil {
		t.Fatal(err)
	}
	if user.Name != "Pong" || user.Email != "pong@example.com" || user.Age != 30 {
		t.Fatalf("unexpected user: %#v", user)
	}
}

func TestMustStructValidationLabels(t *testing.T) {
	t.Parallel()
	schema := MustStruct[createUserRequest]()
	_, err := Parse(schema, `{"name":"x","email":"plain","age":10}`)
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
	byPath := map[string]Issue{}
	for _, issue := range verr.Issues {
		byPath[issue.Path.String()] = issue
	}
	if issue, ok := byPath["name"]; !ok || issue.Label != "姓名" {
		t.Fatalf("name issue = %#v", issue)
	}
	if issue, ok := byPath["email"]; !ok || issue.Label != "邮箱" {
		t.Fatalf("email issue = %#v", issue)
	}
	if issue, ok := byPath["age"]; !ok || issue.Label != "年龄" {
		t.Fatalf("age issue = %#v", issue)
	}
}

func TestStructOptionalOmitEmpty(t *testing.T) {
	t.Parallel()
	type profile struct {
		Name string `json:"name" shape:"trim,nonempty"`
		Nick string `json:"nick,omitempty" shape:"trim,max=20"`
		Bio  string `json:"bio" shape:"optional,max=100"`
	}
	schema := MustStruct[profile]()
	got, err := schema.Parse(map[string]any{"name": " Pong "})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Pong" || got.Nick != "" || got.Bio != "" {
		t.Fatalf("unexpected profile: %#v", got)
	}
}

func TestStructNestedAndSlice(t *testing.T) {
	t.Parallel()
	type address struct {
		City string `json:"city" shape:"trim,nonempty,label='城市'"`
	}
	type payload struct {
		Tags    []string `json:"tags" shape:"min=1,max=3"`
		Address address  `json:"address"`
	}
	schema := MustStruct[payload]()
	got, err := Parse(schema, `{"tags":["a","b"],"address":{"city":" Shanghai "}}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "a" || got.Address.City != "Shanghai" {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestStructDurationAndTimeCoerce(t *testing.T) {
	t.Parallel()
	type job struct {
		Timeout time.Duration `json:"timeout" shape:"coerce"`
		Start   time.Time     `json:"start" shape:"coerce"`
	}
	schema := MustStruct[job]()
	got, err := Parse(schema, `{
		"timeout":"3s",
		"start":"2024-01-02T15:04:05Z"
	}`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Timeout != 3*time.Second {
		t.Fatalf("timeout = %v, want 3s", got.Timeout)
	}
	wantStart, err := time.Parse(time.RFC3339, "2024-01-02T15:04:05Z")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Start.Equal(wantStart) {
		t.Fatalf("start = %v, want %v", got.Start, wantStart)
	}
}

func TestStructSchemaCache(t *testing.T) {
	t.Parallel()
	type item struct {
		Name string `json:"name" shape:"trim,min=1"`
	}
	first, err := Struct[item]()
	if err != nil {
		t.Fatal(err)
	}
	second, err := Struct[item]()
	if err != nil {
		t.Fatal(err)
	}
	if len(first.fields) == 0 || len(second.fields) != len(first.fields) {
		t.Fatalf("cached schema fields = %d / %d", len(first.fields), len(second.fields))
	}
	// Same cached field slice backing (Object copies the slice header to a new
	// array at construction; cache stores one ObjectSchema value).
	if &first.fields[0] != &second.fields[0] {
		t.Fatal("expected Struct to reuse cached field definitions")
	}
	strict := MustStruct[item]().Strict()
	if !strict.strict {
		t.Fatal("Strict() should set strict on the returned copy")
	}
	again := MustStruct[item]()
	if again.strict {
		t.Fatal("cache must keep the non-strict base schema")
	}
}

func TestStructStrictUnknownField(t *testing.T) {
	t.Parallel()
	schema := MustStruct[createUserRequest]().Strict()
	_, err := Parse(schema, `{"name":"Pong","email":"pong@example.com","age":30,"extra":1}`)
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
	found := false
	for _, issue := range verr.Issues {
		if issue.Code == CodeUnknownField {
			found = true
		}
	}
	if !found {
		t.Fatalf("issues = %#v, want unknown field", verr.Issues)
	}
}

func TestBindReaderLimitContext(t *testing.T) {
	t.Parallel()
	var user createUserRequest
	err := BindReaderLimitContext(t.Context(), &user, strings.NewReader(
		`{"name":" Pong ","email":"pong@example.com","age":30}`,
	), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if user.Name != "Pong" || user.Email != "pong@example.com" || user.Age != 30 {
		t.Fatalf("unexpected user: %#v", user)
	}
}

func TestBindStrictUnknownField(t *testing.T) {
	t.Parallel()
	var user createUserRequest
	err := Bind(&user, `{"name":"Pong","email":"pong@example.com","age":30,"extra":true}`)
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
}

func TestBindNilDest(t *testing.T) {
	t.Parallel()
	err := Bind[createUserRequest](nil, `{}`)
	if err == nil {
		t.Fatal("expected error")
	}
}
