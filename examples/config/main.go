package main

import (
	"fmt"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/types"
)

type Config struct {
	Port    uint16         `json:"port" shape:"ifzero=8080,min=1"`
	Timeout types.Duration `json:"timeout" shape:"ifzero=30s,min=1s,max=5m"`
	Debug   *bool          `json:"debug" shape:"ifnull=false"`
}

func main() {
	var config Config
	err := shape.BindJSON(&config, []byte(`{"timeout":"1m"}`))
	debug := false
	if config.Debug != nil {
		debug = *config.Debug
	}
	fmt.Printf("port=%d timeout=%s debug=%t error=%v\n", config.Port, config.Timeout, debug, err)
}
