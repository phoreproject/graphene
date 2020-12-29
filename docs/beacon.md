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

#### Example cmd
```bash
grpcurl -plaintext localhost:11782 pb.BlockchainRPC/GetForkData
```

### `GetGenesisTime()`

#### Example cmd
```bash
grpcurl -plaintext localhost:11782 pb.BlockchainRPC/GetGenesisTime
```

#### Example output
```json
{
  "GenesisTime": "1609268466"
}
```

### `GetLastBlockHash()`

#### Example cmd
```bash
grpcurl -plaintext localhost:11782 pb.BlockchainRPC/GetLastBlockHash
```

#### Example output
```json
{
  "Hash": "APx2Pjd8BCVHNhKnDdIOqrpktJTA/Vw0Cdcg6+ZbMGM="
}
```

### `GetListeningAddresses()`

#### Example cmd
```bash
grpcurl -plaintext localhost:11782 pb.BlockchainRPC/GetListeningAddresses
```

#### Example output
```json
{
  "Addresses": [
    "/ip4/127.0.0.1/tcp/11781/ipfs/12D3KooWK97tVaeUteHFix7S75KteszDcuHPAP4YEkcLjxnJwKP6",
    "/ip4/127.94.0.1/tcp/11781/ipfs/12D3KooWK97tVaeUteHFix7S75KteszDcuHPAP4YEkcLjxnJwKP6",
    "/ip4/192.168.8.100/tcp/11781/ipfs/12D3KooWK97tVaeUteHFix7S75KteszDcuHPAP4YEkcLjxnJwKP6"
  ]
}
```

### `GetMempool(LastBlockHash: []byte)`

#### Example cmd
```bash
grpcurl -plaintext -d '{"LastBlockHash": "Fra1nu7/fc/L31W21ibb3HPE/av44q4rB+sqF+f1upU="}' localhost:11782 pb.BlockchainRPC/GetMempool
```

### `GetProposerForSlot(Slot: uint64)`

#### Example cmd
```bash
grpcurl -plaintext -d '{"Slot": 152}' localhost:11782 pb.BlockchainRPC/GetProposerForSlot
```

#### Example output
```json
{
  "Proposer": 229
}
```

### `GetSlotNumber()`

#### Example cmd
```bash
grpcurl -plaintext localhost:11782 pb.BlockchainRPC/GetSlotNumber
```

#### Example output
```json
{
  "SlotNumber": "215",
  "BlockHash": "qJR1BlAEs2PDPF/6vqpL9MupXP1XGbN7sQD7+1jiaIw=",
  "TipSlot": "215"
}
```

### `GetState()`

#### Example cmd
```bash
grpcurl -plaintext localhost:11782 pb.BlockchainRPC/GetState
```

#### Example output
```json
{
  "state": {
    "Slot": "366",
    "EpochIndex": "45",
    "GenesisTime": "1609268466",
    "ForkData": {},
    "ValidatorRegistry": [
      {
        "Pubkey": "jK4kbGvRrmdTLQB27OhinH+J1JW0UKxpEzbKwxFTQfGhvd6QHEkJyBdEUkAtvwidD+nG5hCkMKknPjzE/rqY2zCnAU/zcHYhuInKfkGcMQOJYJDSaG5vpc7foX1qHph2",
        "WithdrawalCredentials": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
      },
      ...
    ],
    "ValidatorBalances": [
      "6401933448",
      ...
    ],
    "ValidatorRegistryLatestChangeEpoch": "45",
    "ValidatorRegistryDeltaChainTip": "itP3Ovcg7iSDdzLM/rHuZ4C1yAE4cufhw185L1D2+yU=",
    "RandaoMix": "Zej07CZfAVzvGqatpxO+zBzKB0Jf6a57SVBdB9PcBLc=",
    "NextRandaoMix": "lC3A4TszZs+wFht07aLPQlxrWp50AhSd06ZtHE75j40=",
    "ShardCommittees": [
      {
        "Committees": [
          {
            "Committee": [
              24,
              144,
              171,
              ...
            ]
          }
        ]
      },
      {
        "Committees": [
          {
            "Shard": "1",
            "Committee": [
              70,
              82,
              162,
              ...
            ]
          }
        ]
      },
      {
        "Committees": [
        ]
      },
      ...
    ],
    "PreviousJustifiedEpoch": "43",
    "JustifiedEpoch": "44",
    "JustificationBitField": "35184372088831",
    "FinalizedEpoch": "43",
    "LatestCrosslinks": [
      {
        "Slot": "353",
        "ShardBlockHash": "ED43/YoEIG0ZEHXtwkximprCMXAC2nZY6GXrl6U4Mpk=",
        "ShardStateHash": "oaSVkX+xLeg7ynMqi/bJUKl81ZgfwbPU5eDceQ+RUpk="
      },
      ...
    ],
    "PreviousCrosslinks": [
      {
        "Slot": "345",
        "ShardBlockHash": "V+PhZ5gTa764rnwVLkIt23dYYpyvz0S0jtpGkBwnuBg=",
        "ShardStateHash": "oaSVkX+xLeg7ynMqi/bJUKl81ZgfwbPU5eDceQ+RUpk="
      },
      ...
    ],
    "ShardRegistry": [
      "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
      ...
    ],
    "LatestBlockHashes": [
      "eQ6PtrWL5yf0bsiaeadAo94MOWeeemLriF/qhhqmX6A=",
      "jPX0zcItNy3fsJiZTTQWmSxb+9i/9Rf/1tlTt7LqvuY=",
      ...
    ],
    "CurrentEpochAttestations": [
      {
        "Data": {
          "Slot": "361",
          "BeaconBlockHash": "a6M08ERitz+tCi/G5SxmhnLscnsGJJ+rK4YW1XgpVk4=",
          "TargetEpoch": "45",
          "TargetHash": "HJoAij8eKwT1sP+RdP2I/Cv0lAJE+xPno7WG58l86TM=",
          "SourceEpoch": "44",
          "SourceHash": "4iHh0oHEzWzyZMBXAPvOaP2Kp/fWQlrxFdB9npRzpnA=",
          "ShardBlockHash": "3W2ZguujWfInNtXsFQRIQrS1DVX1lYK0iLS9/FITCB4=",
          "LatestCrosslinkHash": "ED43/YoEIG0ZEHXtwkximprCMXAC2nZY6GXrl6U4Mpk=",
          "ShardStateHash": "oaSVkX+xLeg7ynMqi/bJUKl81ZgfwbPU5eDceQ+RUpk="
        },
        "ParticipationBitfield": "/////w==",
        "CustodyBitfield": "AAAAAA==",
        "InclusionDelay": "2",
        "ProposerIndex": 121
      },
      ...
    ],
    "PreviousEpochAttestations": [
      {
        "Data": {
          "Slot": "353",
          "BeaconBlockHash": "uTfE07hkjhLG2+M0hLLitSZG+jh3wV/92EjxY4qhO48=",
          "TargetEpoch": "44",
          "TargetHash": "4iHh0oHEzWzyZMBXAPvOaP2Kp/fWQlrxFdB9npRzpnA=",
          "SourceEpoch": "43",
          "SourceHash": "5vWauQ9I+s7cr3bhvi2Z0+/xI42B7PRGZ3DyHGbc+oA=",
          "ShardBlockHash": "ED43/YoEIG0ZEHXtwkximprCMXAC2nZY6GXrl6U4Mpk=",
          "LatestCrosslinkHash": "V+PhZ5gTa764rnwVLkIt23dYYpyvz0S0jtpGkBwnuBg=",
          "ShardStateHash": "oaSVkX+xLeg7ynMqi/bJUKl81ZgfwbPU5eDceQ+RUpk="
        },
        "ParticipationBitfield": "/////w==",
        "CustodyBitfield": "AAAAAA==",
        "InclusionDelay": "2",
        "ProposerIndex": 142
      },
      ...
    ]
  }
}

```

### `GetStateRoot(BlockHash: []byte)`

#### Example cmd
```bash
grpcurl -plaintext -d '{"BlockHash": "VczYbC8KiBQIxNJ+2HMoCHntq0cioh7e5DMzrOOf+DU="}' localhost:11782 pb.BlockchainRPC/GetStateRoot
```

#### Example output
```json
{
  "StateRoot": "joyx0GtBc2THt9wG4QT1Esub+uJsS7yBaJf/rpywE4M="
}
```

### `GetValidatorInformation(ID: uint32)`

#### Example cmd
```bash
grpcurl -plaintext -d '{"ID": 11}' localhost:11782 pb.BlockchainRPC/GetValidatorInformation
```

#### Example output
```json
{
  "Pubkey": "hhvIvotvzzg6hibvJSzd/KKvYfJD5K4OoM/EJLNq4QnsB9fn3J5fQuLlynphItXjGCy5aLpHnnk44K40WZIc62D0+4pGdeCtYIh6aa8vQmcwaXnJGRAXJURuTDtWiaDm",
  "WithdrawalCredentials": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
}
```

### `GetValidatorProof(ValidatorID: uint32)`

#### Example cmd
```bash
grpcurl -plaintext -d '{"ValidatorID": 11}' localhost:11782 pb.BlockchainRPC/GetValidatorProof
```

#### Example output
```json
{
  "Proof": {
    "ShardID": "3",
    "ValidatorIndex": "5",
    "PublicKey": "hhvIvotvzzg6hibvJSzd/KKvYfJD5K4OoM/EJLNq4QnsB9fn3J5fQuLlynphItXjGCy5aLpHnnk44K40WZIc62D0+4pGdeCtYIh6aa8vQmcwaXnJGRAXJURuTDtWiaDm",
    "Proof": {
      "Key": "okOpuBMphJ0pD0aUSeVZBHZbAHuMUv8R5b00nEogUmY=",
      "Value": "6vzEJgaoy7Kt2dZhSxdbG9uTJ06YeekBwlM17iKh+Rw=",
      "WitnessBitfield": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQP0=",
      "WitnessHashes": [
        "yM6fvIQVphO9vVaA643/PAjXfGhLpbbRLl1Vnu+FP7Y=",
        "U+f1lpbB582Ki199dRFh64wDh7OYOVCjeBcyum7ntQU=",
        "Ls8TUWTNl7Y5NOTu0qyhmQT6HzExtM5ny98KU2zXlIs=",
        "g2F/2RokI+6/35zZ7JFBilH51dw4tYlkRBWDN4T6wyQ=",
        "3YzkLWJS+IKXnr8m41Okn4wKL2FjqUD0xwkIGAJD+N8=",
        "WsWxNB+q6JGh72kIW0oKekqtK/IX77GfpSzURp7Qjzo=",
        "b621dHKA23jfE3Uel+wivJouKqI4kfJr4O6sf5XcfGE=",
        "T4cARhAi9PBgKT9hEjDxtZm9Ux9/BGzqDpSUcr7Q4As="
      ],
      "LastLevel": 245
    }
  },
  "FinalizedHash": "0r3fc4tAuboEBCcThj5hwawK8vN1wm9N6nFfYQvl/04="
}
```

### `GetValidatorRoot(BlockHash: []byte)`

#### Example cmd
```bash
grpcurl -plaintext -d '{"BlockHash": "aDvz7a++i40GWnIE37tqjVUQ5Aei60yAG7Z8JKuajfw="}' localhost:11782 pb.BlockchainRPC/GetValidatorRoot
```

#### Example output
```json
{
  "ValidatorRoot": "KYGT51k86npND92GAeLLvD2YXzmPKryAvTvDROQ/n+Y="
}
```

### `SubmitAttestation(Attestation: Attestation)`

### `SubmitBlock(Block: *Block)`
