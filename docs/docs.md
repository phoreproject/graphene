# Phore Synapse Documentation

This documentation gives information about how each of the different modules interact and a human-readable overview about how Phore Synapse works. Design choices and information will also be documented here. The documentation is not complete at this time, but will be filled in over time with more information.

## Table of Contents

1. [Overview](overview.md)
  a. [Overview](overview.md#overview)
  b. [Beacon Chain](overview.md#beacon-chain)
  c. [Attestations](overview.md#attestations)
  d. [Shard/Block Assignments](overview.md#shard-block-assignments)
  e. [Entropy](overview.md#entropy)
  f. [Slashings](overview.md#slashings)
  g. [Rewards](overview.md#rewards)
2. [Block Processing](block-processing.md)
3. [Mempool](mempool.md)
  a. [Validation](mempool.md#validation)
  b. [Prioritization](mempool.md#prioritization)
  c. [Limits](mempool.md#limits)
4. State Management
5. BLS Signatures
6. Modules
  a. Beacon
  b. [Validator](validator.md)
  c. Shard
  d. Explorer
7. Communication
  a. P2P
  b. Beacon-Validator
  c. Beacon-Shard
