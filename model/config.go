package model

type ConfigSpec struct {
	SpecDir string `yaml:"spec_dir"`
	// Gas
	// Limits
	// Stuff
}

var DefaultConfigSpec = &ConfigSpec{}

func (spec *ConfigSpec) Validate() bool {
	return true
}
