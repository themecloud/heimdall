package config

import (
	"fmt"

	"github.com/themecloud/heimdall"
	"github.com/themecloud/heimdall/outputs"
)

// TODO
type Config struct {
	Output *heimdall.RunningOutput
}

// TODO
func NewConfig(outputName string) *Config {
	creator, ok := outputs.Outputs[outputName]
	if !ok {
		return &Config{}
	}
	output := creator()
	oc := heimdall.NewOutputConfig(outputName)

	ro := heimdall.NewRunningOutput(outputName, output, oc)

	c := &Config{
		Output: ro,
	}
	return c
}

// AddOutput TODO
func (c *Config) AddOutput(name string) error {
	creator, ok := outputs.Outputs[name]
	if !ok {
		return fmt.Errorf("Undefined but requested output: %s", name)
	}
	output := creator()
	oc := heimdall.NewOutputConfig(name)

	ro := heimdall.NewRunningOutput(name, output, oc)
	c.Output = ro
	return nil
}
