package config

import (
	"path/filepath"
	"runtime"

	homedir "github.com/mitchellh/go-homedir"
)

func defaultDataPath() (path string) {
	if runtime.GOOS == "darwin" {
		return "~/Library/Application Support"
	}
	return "~"
}

// GetBaseDirectory gets the directory for synapse
func GetBaseDirectory(testModeEnabled bool) (path string, err error) {
	path, err = homedir.Expand(filepath.Join(defaultDataPath(), directoryName(testModeEnabled)))
	if err == nil {
		path = filepath.Clean(path)
	}
	return path, err
}

func directoryName(isTestnet bool) (directoryName string) {
	if runtime.GOOS == "linux" {
		directoryName = ".phore-synapse"
	} else {
		directoryName = "phore-synapse"
	}

	if isTestnet {
		directoryName += "-testnet"
	}
	return directoryName
}
