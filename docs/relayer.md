# Relayer Module

The relayer module is responsible for receiving transactions from users, packaging them up, and providing the packages to validators. Eventually, they'll receive a set fee for their services because they have to store the entire state and provide the state to validators.

## Table of Contents

1. [Core Functionality](#core-functionality)
	1. [Storing State](#storing-state)
	2. [Generating Packages](#generating-packages)
	3. [Receiving Transactions](#receiving-transactions)
2. [P2P Interface](#p2p-interface)
	1. [Protocol Information](#protocol-information)
	2. [`GetPackage`](#getpackage)
	3. [`SubmitTransaction`](#submittransaction)
4. [RPC Interface (TBD)](#rpc-interface)

## Core Functionality

The relayer module has three main jobs: store state, receive incoming transactions, and generate transaction packages.

### Storing State

The relayer module must store the full state for the shards it is tracking. To do so, it must store the state on disk and load it to memory when needed.

### Generating Packages

The relayer must maintain a shard chain and generate transaction packages based on the current tip of the shard chain.

### Receiving Transactions

The relayer module must maintain a mempool for each shard it is tracking.

## P2P Interface

The relayer module has a main P2P interface used for receiving incoming transactions, sharing shard transactions with other relayers interested in the same shard, and generating transaction packages.

### Protocol Information

The relayer module will implement the interface: `/phore/relayer/shard/{N}` where `{N}` represents a specific shard the relayer will handle. The relayer module will implement this protocol for **each** shard it is currently tracking.

### `GetPackage`

```go
type GetPackageRequest struct {
	StateRoot chainhash.Hash
}

type GetPackageResponse struct {
	HasPackage bool
	Package execution.Package
}
```

`GetPackage` gets a transaction package for a specific ShardID and tip state root. If the relayer has the same state root, then the relayer returns a transaction package which includes transactions, witnesses, and the new state root. If the relayer does not have the same state root, it designates that in the response.

### `SubmitTransaction`

```go
type SubmitTransactionRequest struct {
	Transaction execution.Transaction
}

type SubmitTransactionResponse struct {
	Success bool
}
```

`SubmitTransaction` submits a transaction to the relayer mempool. The relayer module sends a success flag if the transaction returned 0. The transaction should be added to the mempool if it was successful.

## RPC Interface

TBD
