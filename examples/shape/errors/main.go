package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/validate"
)

type Request struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

var requestSchema = shape.New[Request](
	shape.String("Name").NotEmpty().MinLength(3).Label("姓名"),
	shape.Int("Age").Min(18).Label("年龄"),
)

func main() {
	validate.SetLanguage(validate.English)
	english := requestSchema.Validate(Request{})
	chinese := validate.Localize(english, validate.SimplifiedChinese)
	fmt.Println("current language:", validate.CurrentLanguage())
	fmt.Println("localized copy:", chinese)

	ctx := validate.WithLocale(context.Background(), validate.SimplifiedChinese)
	err := requestSchema.ValidateContext(ctx, Request{})
	err = validate.LocalizeContext(ctx, err)
	fmt.Println("context language:", validate.LocaleFromContext(ctx))

	var validationError *validate.Error
	if errors.As(err, &validationError) {
		for _, issue := range validationError.Issues {
			fmt.Printf("code=%s path=%s label=%s message=%s\n", issue.Code, issue.Path, issue.Label, issue.Message)
		}
	}

	err = requestSchema.ValidateFirst(Request{})
	if errors.As(err, &validationError) {
		fmt.Println("first issue:", validationError.Issues[0].Path.String())
	}
}
