package cfg

// GlobalOptions are options to be applied globally and set at the root of the config.
type GlobalOptions struct {
	LogLevel   string `yaml:"log_level" cli:"level"`
	ForceColors bool `yaml:"log_colors" cli:"colors"`
	ConfigFile string `cli:"config"`
}
