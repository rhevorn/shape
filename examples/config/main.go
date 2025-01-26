package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rhevorn/shape"
)

type Config struct {
	Port    Port
	Debug   bool
	Timeout time.Duration
}

type Port uint16

var configSchema = shape.Object[Config](
	shape.Field("port", shape.CoerceNumber[Port]().Min(1).Max(65535), func(config *Config, value Port) {
		config.Port = value
	}).Default(Port(8080)),
	shape.Field("debug", shape.CoerceBool(), func(config *Config, value bool) {
		config.Debug = value
	}).Default(false),
	shape.Field("timeout", shape.CoerceDuration(), func(config *Config, value time.Duration) {
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
