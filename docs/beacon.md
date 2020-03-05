# Beacon Module

This document describes the responsibilities of the beacon chain and documentation about the RPC for the beacon module.

## Table Of Contents

1. [RPC](#rpc)

## RPC

This section explains each RPC method and the return value for each.

### `GetEpochInformation(EpochIndex: uint64)`

This function gets epoch information for the specified epoch. It includes all of the information needed for validators to propose and attest blocks.

It returns a wrapper message that can either contain epoch information or signal that the client doesn't have enough information (as is the case when syncing the blockchain).

Specficially, if the client does have epoch information, the function returns the following values:

- `ShardCommitteesForSlots` - array of committees for each slot; slots start at the previous epoch and end at the end of the epoch requested
- `Slot` - the earliest slot of the `ShardCommitteesForSlots` array
- `TargetHash` - the hash at the start of the epoch requested (`epoch_index * epoch_length`)
- `JustifiedEpoch` - the current justified epoch
- `JustifiedHash` - the hash at the beginning of the current justified epoch (`JustifiedEpoch * epoch_length`)
- `LatestCrosslinks` - the current crosslinks from the current justified epoch
- `PreviousTargetHash` - the hash at the start of the previous epoch
- `PreviousJustifiedEpoch` - the previous justified epoch
- `PreviousJustifiedHash` - the hash at the beginning of the previous justified epoch (`PreviousJustifiedEpoch * epoch_length`)
- `PreviousCrosslinks` - the crosslinks from the previous justified epoch

Information about how the validator uses these values to construct valid attestations and block proposals can be found in the [validator section](validator.md) of the docs.