# Beacon Module

This document describes the responsibilities of the beacon chain and documentation about the RPC for the beacon module.

## Table Of Contents

1. [RPC](#rpc)

## RPC

This section explains each RPC method and the return value for each.

### `GetBlock(Hash: []byte)`

#### Example cmd:
```bash
grpcurl -plaintext -d '{"Hash": "Fra1nu7/fc/L31W21ibb3HPE/av44q4rB+sqF+f1upU="}' localhost:11782 pb.BlockchainRPC/GetBlock
```

#### Example output:
```json
{
  "Block": {
    "Header": {
      "SlotNumber": "6299",
      "ParentRoot": "gv54SdGlfPGO/hbRZWottVi47Qxad8DSuk7hCX2jkMc=",
      "StateRoot": "riDeAsBg72+E7MdoUVexTqpsOSqhgkPvmp3pr2eytNo=",
      "RandaoReveal": "hatEc7p6w4Tpnq7McOS9mo0AH4cTx7AYLsOAFd6u7Uh3Ie1R7zw631C5o2nO4TMz",
      "Signature": "t19+qSPhOO0bfjprskuOO7dz2XdEgtwXcxIHB0jgzUR360CUUerBvsCLja97fP2+",
      "ValidatorIndex": 225
    },
    "Body": {
      "Attestations": [
        {
          "Data": {
            "Slot": "6297",
            "BeaconBlockHash": "2eX5M1FpKr89071CGZt1QbPMZiwX2kf24Ft639+4brg=",
            "TargetEpoch": "787",
            "TargetHash": "BYqCl4+af9G14jt54AUtsVTT3En1C5fRpfQElOerhcM=",
            "SourceEpoch": "786",
            "SourceHash": "nmcGsOdZ+ukr59bK4It3Z/w4XRC8WnLCKwtqoUxmlhM=",
            "ShardBlockHash": "hz8kDABC/B9yknLlGSZ99p1hA+bMUooDzqgK44MmNg8=",
            "LatestCrosslinkHash": "ZVeMezZzTm/nj55mPM090HSh/kSbtnXkZGao6GlnWwM=",
            "ShardStateHash": "oaSVkX+xLeg7ynMqi/bJUKl81ZgfwbPU5eDceQ+RUpk="
          },
          "ParticipationBitfield": "/////w==",
          "CustodyBitfield": "AAAAAA==",
          "AggregateSig": "pxCkLixJvZ7dAxngeB3X8GxBB1qDJrJqIMBUTHgUt30ZVPhe8xJ0zlHAEEypy74T"
        }
      ]
    }
  }
}
```

### `GetBlockHash(SlotNumber: uint64)`

#### Example cmd:
```bash
grpcurl -plaintext -d '{"SlotNumber": 123}' localhost:11782 pb.BlockchainRPC/GetBlock
```

#### Example output:
```json
{
  "Hash": "CNFMesoCJemNuE5EuZKAqBzhznfSSt8l8DQ7KwcfypU="
}
```

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

### `GetForkData()`

### `GetGenesisTime()`

### `GetLastBlockHash()`

### `GetListeningAddresses()`

### `GetMempool(LastBlockHash: []byte)`

### `GetProposerForSlot(Slot: uint64)`

### `GetSlotNumber()`

### `GetState()`

### `GetStateRoot(BlockHash: []byte)`

### `GetValidatorInformation(ID: uint32)`

### `GetValidatorProof(ValidatorID: uint32)`

### `GetValidatorRoot(BlockHash: []byte)`

### `SubmitAttestation(Attestation: Attestation)`

### `SubmitBlock(Block: *Block)`
