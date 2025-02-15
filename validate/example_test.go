package validate_test

import (
	"errors"
	"fmt"

	"github.com/rhevorn/shape/validate"
)

func ExampleString() {
	username := validate.String().NotEmpty().MinLength(3)
	err := username.Validate("")

	var validationError *validate.Error
	if errors.As(err, &validationError) {
		for _, issue := range validationError.Issues {
			fmt.Println(issue.Code, issue.Message)
		}
	}
	// Output:
	// invalid_value must not be empty
	// too_small must contain at least 3 characters
}
