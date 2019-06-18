# Phore Synapse Documentation

This documentation gives information about how each of the different modules interact and a human-readable overview about how Phore Synapse works. Design choices and information will also be documented here. The documentation is not complete at this time, but will be filled in over time with more information.

## Table of Contents


1. [Overview](overview.md)
    1. [Overview](overview.md#overview)
    2. [Beacon Chain](overview.md#beacon-chain)
    3. [Attestations](overview.md#attestations)
    4. [Shard/Block Assignments](overview.md#shard-block-assignments)
    5. [Entropy](overview.md#entropy)
    6. [Slashings](overview.md#slashings)
    7. [Rewards](overview.md#rewards)
2. [Block Processing](block-processing.md)

    1. Slot Transition
    2. [Block Transition](block-transition.md)
    3. Epoch Transition
3. [Mempool](mempool.md)
    1. [Validation](mempool.md#validation)
    2. [Prioritization](mempool.md#prioritization)
    3. [Limits](mempool.md#limits)
4. State Management
5. BLS Signatures
6. Modules
    1. Beacon
    2. [Validator](validator.md)
    3. Shard
    4. Explorer
7. Communication
    1. P2P
    2. Beacon-Validator
    3. Beacon-Shard
