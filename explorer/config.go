package explorer

import "github.com/phoreproject/synapse/beacon/config"

// Config is the explorer app config.
type Config struct {
	DataDirectory string
	NetworkConfig *config.Config
}
