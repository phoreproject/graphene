# Phore Synapse

[![Build Status](https://travis-ci.com/phoreproject/synapse.svg?branch=master)](https://travis-ci.com/phoreproject/synapse) [![codecov](https://codecov.io/gh/phoreproject/synapse/branch/master/graph/badge.svg)](https://codecov.io/gh/phoreproject/synapse)

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