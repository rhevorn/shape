package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/rhevorn/shape"
	shapetypes "github.com/rhevorn/shape/types"
)

type Metadata struct{ Source string }

type Settings struct {
	Enabled  bool                `json:"enabled"`
	Attempts uint8               `json:"attempts"`
	Total    int64               `json:"total"`
	Ratio    float64             `json:"ratio"`
	Created  time.Time           `json:"created"`
	Timeout  shapetypes.Duration `json:"timeout"`
	Metadata Metadata            `json:"metadata"`
}

var settingsSchema = shape.New[Settings](
	shape.Bool("Enabled"),
	shape.Number[uint8]("Attempts").Between(1, 5),
	shape.Int64("Total").NonNegative(),
	shape.Float64("Ratio").Between(0, 1),
	shape.Time("Created").Refine(func(value time.Time) error {
		if value.IsZero() {
			return errors.New("created time is required")
		}
		return nil
	}),
	shape.Duration("Timeout").IfZero(shapetypes.Duration(30*time.Second)).Positive(),
	shape.Value[Metadata]("Metadata").Apply(func(value Metadata) (Metadata, error) {
		if value.Source == "" {
			value.Source = "api"
		}
		return value, nil
	}),
)

func main() {
	value := Settings{
		Enabled: true, Attempts: 3, Ratio: 0.5,
		Created: time.Now(), Timeout: shapetypes.Duration(time.Minute),
	}
	value, err := settingsSchema.Transform(value)
	if err == nil {
		err = settingsSchema.Validate(value)
	}
	fmt.Printf("settings: %#v error=%v\n", value, err)
}
