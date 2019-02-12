# Phore Synapse

[![Build Status](https://travis-ci.com/phoreproject/synapse.svg?branch=master)](https://travis-ci.com/phoreproject/synapse) [![codecov](https://codecov.io/gh/phoreproject/synapse/branch/master/graph/badge.svg)](https://codecov.io/gh/phoreproject/synapse) [![Go Report Card](https://goreportcard.com/badge/github.com/phoreproject/synapse)](https://goreportcard.com/report/github.com/phoreproject/synapse)

A proof-of-stake, sharded blockchain built from scratch.

This is loosely based on the Ethereum sharding system.

## TODO List

- implement BLS sigs
- implement serialization of active/crystallized states into merkle roots
- registration logic

## Service Port List

- `11781` - P2P network default port
- `11782` - beacon chain RPC port
- `11783` - P2P service RPC port

## Testing

```bash
make test
```

Also, to test validator code, run the following commands in 3 separate terminals:

```bash
go run cmd/p2p/synapsep2p.go
go run cmd/beacon/synapsebeacon.go
go run cmd/validator/synapsevalidator.go -validators 0-4095
```

## Building

```bash
make build
```

## Installing pre-commit checks

```bash
pip install precommit
precommit install
```
