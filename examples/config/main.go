package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rhevorn/goshape"
)

type Config struct {
	Port    int
	Debug   bool
	Timeout time.Duration
}

var configSchema = goshape.Object[Config](
	goshape.Field("port", goshape.CoerceInt().Min(1).Max(65535), func(config *Config, value int) {
		config.Port = value
	}).Default(8080),
	goshape.Field("debug", goshape.CoerceBool(), func(config *Config, value bool) {
		config.Debug = value
	}).Default(false),
	goshape.Field("timeout", goshape.CoerceDuration(), func(config *Config, value time.Duration) {
		config.Timeout = value
	}).Default(5*time.Second),
).Strict()

func main() {
	input := map[string]any{}
	if value, ok := os.LookupEnv("PORT"); ok {
		input["port"] = value
	}
	if value, ok := os.LookupEnv("DEBUG"); ok {
		input["debug"] = value
	}
	if value, ok := os.LookupEnv("TIMEOUT"); ok {
		input["timeout"] = value
	}

	config, err := configSchema.Parse(input)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", config)
}
