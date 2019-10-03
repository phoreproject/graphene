package config

// Options are the options passed to the module.
type Options struct {
	ShardRPC string `yaml:"shard_addr" cli:"shard"`
	P2PListen string `yaml:"p2p_listen_addr" cli:"listen"`
	RPCListen string `yaml:"rpc_listen_addr" cli:"rpclisten"`
	Shards []string `yaml:"shards" cli:"shards"`
}