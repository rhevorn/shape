package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/validate"
)

type Request struct {
	Name   string         `json:"name"`
	Age    int            `json:"age"`
	Scores map[string]int `json:"scores"`
}

var requestSchema = shape.New[Request](
	shape.Field("Name", shape.String().NotEmpty().MinLength(3).Label("display name")),
	shape.Field("Age", shape.Int().Min(18).Label("age")),
	shape.Field("Scores", shape.Map(shape.String().NotEmpty(), shape.Int().NonNegative())),
)

func main() {
	err := requestSchema.Validate(Request{Scores: map[string]int{"math": -1}})
	printValidation("all", err)

	err = requestSchema.ValidateFirst(Request{Scores: map[string]int{"math": -1}})
	printValidation("first", err)

	validate.SetLanguage(validate.SimplifiedChinese)
	printValidation("global zh-CN", requestSchema.ValidateFirst(Request{}))

	english := validate.WithLocale(context.Background(), validate.English)
	printValidation("request English", requestSchema.ValidateFirstContext(english, Request{}))

	failingTransform := shape.New[Request](
		shape.Field("Name", shape.String().Apply(func(string) (string, error) {
			return "", errors.New("normalizer unavailable")
		})),
	)
	_, err = failingTransform.Transform(Request{Name: "Pong"})
	var transformError *shape.TransformError
	if errors.As(err, &transformError) {
		fmt.Printf("transform: path=%s cause=%v unwrap=%v\n",
			transformError.Path,
			transformError.Err,
			errors.Unwrap(transformError),
		)
	}

	path := validate.Path{
		validate.FieldPath("scores"),
		validate.MapKeyPath(3),
		validate.IndexPath(1),
	}
	encoded, _ := json.Marshal(path)
	fmt.Printf("path: text=%s json=%s\n", path, encoded)
}

func printValidation(label string, err error) {
	var validationError *validate.Error
	if !errors.As(err, &validationError) {
		fmt.Printf("%s: %v\n", label, err)
		return
	}
	fmt.Printf("%s: %d issue(s)\n", label, len(validationError.Issues))
	for _, issue := range validationError.Issues {
		fmt.Printf("  code=%s path=%s label=%q expected=%v received=%v message=%q\n",
			issue.Code,
			issue.Path,
			issue.Label,
			issue.Expected,
			issue.Received,
			issue.Message,
		)
	}
}
