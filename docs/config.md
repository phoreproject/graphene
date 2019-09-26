# Configuration

This document details the configuration system used for Synapse. All configuration settings can be configured straight from the command line, or set via a YAML file, loaded as part of the `ConfigFile` global variable.

## Table of Contents

- [Overview](#overview)
- [Interfaces](#interfaces)
- [Global Variables](#global-variables)
- [Module-Specific Variables](#module-specific-variables)
    - [Beacon](#beacon)
    - [Validator](#validator)
    - [Shard](#shard)

## Overview

Global variables are declared at the root of all configuration files. These configuration variables affect how the program runs at a global level. Module-specific variables are configuration for how modules in the program run. For example, the beacon module might have a configuration option for the time of the genesis block. An example of a global variable might be the log-level for the program.

## Interfaces

```go
type ConfigInterface struct {
	Get(key string) interface{} // returns the config key provided or nil
	GetGlobal(key string) interface{} // returns the global config key or nil if it does not exist or it a local variable.
}
```

## Global Variables

|Name|CLI|YAML|Type|Description
|----|---|----|----|-----------|
|Log Level|`level`|`log_level`|string|level of log verbosity|
|Config File|N/A|`config`|string|path of config file|

## Module-specific Variables

### Beacon

|Name|CLI|YAML|Type|Description
|----|---|----|----|-----------|
|RPC Listen Address|`rpclisten`|`rpc_listen_addr`|string|multiaddr to listen on for RPC|
|Chain Config File|`chaincfg`|`chain_config`|string|ID of the chain config to use. IDs specified in `beacon/config.go`|
|Resync|`resync`|`resync`|bool|should the beacon chain resync the chain|
|Data Directory|`datadir`|`data_directory`|string|path of the data directory to use|
|Genesis Time|`genesistime`|`genesis_time`|string|time of the genesis block (either: `+20`, `-20`, or a specific timestamp)|
|Initial Connections|`connect`|`initial_connections`|comma-separated string|list of multiaddr's separated by commas to connect to initially|
|P2P Listen Address|`listen`|`p2p_listen_addr`|string|multiaddr to listen on for P2P

### Validator

|Name|CLI|YAML|Type|Description
|----|---|----|----|-----------|
|Beacon RPC Address|`beacon`|`beacon_addr`|string|multiaddr of beacon module RPC
|Shard RPC Address|`shard`|`shard_addr`|string|multiaddr of shard module RPC
|Validators To Run|`validators`|`validators`|comma-separated string|range of ints or list of ints of IDs to run validators for|
|Root Key|`rootkey`|`root_key`|string|root key of validators keys|
|Validator Listen Port|`listen`|`listen_addr`|string|address for validator to listen on|
|Network ID|`networkid`|`network_id`|string|network ID to run validator on|

### Shard
|Name|CLI|YAML|Type|Description
|----|---|----|----|-----------|
|Beacon RPC Address|`beacon`|`beacon_addr`|string|multiaddr of beacon chain RPC|
|RPC Listen Port|`listen`|`listen_addr`|string|multiaddr to listen on for RPC|
