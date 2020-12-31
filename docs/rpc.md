# RPC specification

This document explains how to use RPC commands to communicate with modules.

## Each module have it's own rpc section with particular method descriptions.
* Modules
    1. [Beacon](validator.md)
    2. [Validator](validator.md)
    3. [Shard](shard.md)
    4. Explorer
    5. [Config](config.md)
    
    
## Each module use Google grpc-go server implementation. [Github](https://github.com/grpc/grpc-go)

## Using RPC:

### I. Connecting from web.
Use JS [library](https://github.com/grpc/grpc-web) provided by Google.

### II. Using command line. 
The easies way is to download [this](https://github.com/fullstorydev/grpcurl) package.

#### Examples:
Print all available methods for beacon chain.
```bash
grpcurl -plaintext localhost:11782 list pb.BlockchainRPC
```

Print all beacon chain methods with params and return types.
```bash
grpcurl -plaintext localhost:11782 describe pb.BlockchainRPC
```

Print all available methods for shards chain.
```bash
grpcurl -plaintext localhost:11783 describe pb.ShardRPC
```

Print all shard chain methods with params and return types.
```bash
grpcurl -plaintext localhost:11783 describe pb.ShardRPC
```