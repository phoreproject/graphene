package config

// Options are bare options passed to the explorer app.
type Options struct {
	BeaconRPC     string `yaml:"beacon_addr" cli:"beacon"`
	NetworkID     string `yaml:"network_id" cli:"networkid"`
	DataDirectory string `yaml:"data_directory" cli:"datadirectory"`
	ChainCFG      string `yaml:"chain_config" cli:"chaincfg"`
}
