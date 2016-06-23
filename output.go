package heimdall

// Output TODO
type Output interface {
	// Description returns a one-sentence description on the Output
	Description() string
	// SampleConfig returns the default configuration of the Output
	SampleConfig() string
	// Write takes in group of points to be written to the Output
	Send(mails []Mail) error
}

// OutputConfig containing name and filter
type OutputConfig struct {
	Name string
}

// RunningOutput contains the output configuration
type RunningOutput struct {
	Name   string
	Output Output
	Config *OutputConfig
}

// NewOutputConfig TODO
func NewOutputConfig(name string) *OutputConfig {
	oc := &OutputConfig{
		Name: name,
	}
	return oc
}

// NewRunningOutput TODO
func NewRunningOutput(name string, output Output, conf *OutputConfig) *RunningOutput {
	ro := &RunningOutput{
		Name:   name,
		Output: output,
		Config: conf,
	}
	return ro
}
