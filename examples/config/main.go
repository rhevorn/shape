package main

import (
	"fmt"
	"time"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/types"
)

type Config struct {
	Environment string         `json:"environment" shape:"ifzero=production,trim,oneof=development|staging|production"`
	Port        uint16         `json:"port" shape:"ifzero=8080,min=1"`
	Timeout     types.Duration `json:"timeout" shape:"ifzero=30s,min=1s,max=5m"`
	Debug       *bool          `json:"debug" shape:"ifnull=false"`
}

var configSchema = shape.Struct[Config]()

var strict = shape.JSONOptions{
	DisallowUnknownFields: true,
	MaxBytes:              64 << 10,
}

func main() {
	config, err := configSchema.ParseJSON([]byte(`{"timeout":"1m"}`), strict)
	fmt.Printf("parsed: env=%s port=%d timeout=%s debug=%t error=%v\n",
		config.Environment, config.Port, config.Timeout, boolean(config.Debug), err)

	target := Config{Environment: "development", Port: 3000, Timeout: types.Duration(time.Minute)}
	err = shape.BindJSON(&target, []byte(`{"environment":"invalid","timeout":"0s"}`), strict)
	fmt.Printf("invalid bind kept old value: env=%s port=%d timeout=%s error=%v\n",
		target.Environment, target.Port, target.Timeout, err)
}

func boolean(value *bool) bool {
	return value != nil && *value
}
