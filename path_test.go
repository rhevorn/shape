package shape

import (
	"encoding/json"
	"testing"
)

func TestPathString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path Path
		want string
	}{
		{name: "root", want: "$"},
		{name: "nested", path: Path{FieldPath("users"), IndexPath(3), FieldPath("address"), FieldPath("zip")}, want: "users[3].address.zip"},
		{name: "non identifier", path: Path{FieldPath("user-name")}, want: `["user-name"]`},
		{name: "unicode identifier", path: Path{FieldPath("用户"), FieldPath("姓名")}, want: "用户.姓名"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := test.path.String(); got != test.want {
				t.Fatalf("Path.String() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPathMarshalJSON(t *testing.T) {
	t.Parallel()

	path := Path{FieldPath("users"), IndexPath(3), FieldPath("email")}
	got, err := json.Marshal(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := `["users",3,"email"]`; string(got) != want {
		t.Fatalf("json.Marshal(Path) = %s, want %s", got, want)
	}
}

func TestPrefixIssuesDoesNotMutateInput(t *testing.T) {
	t.Parallel()

	original := []Issue{{
		Code:    CodeInvalidEmail,
		Path:    Path{FieldPath("email")},
		Message: "invalid email",
	}}
	prefixed := prefixIssues(original, IndexPath(2))

	if got := original[0].Path.String(); got != "email" {
		t.Fatalf("prefixIssues mutated input path: %q", got)
	}
	if got := prefixed[0].Path.String(); got != "[2].email" {
		t.Fatalf("prefixed path = %q, want %q", got, "[2].email")
	}
}
