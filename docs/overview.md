# Phore graphene

Phore graphene is an implementation of a sharded blockchain built on top of the Phore ecosystem. This document will provide a simple explanation of the overall architecture, security features, and implementation details.

## Overview

Phore graphene consists of one main chain and many shard chains. The main chain coordinates shard chains by finalizing shard blocks (using CASPER). Funds can transfer between shards by locking funds on one shard, finalizing the lock transaction on the beacon chain, and unlocking funds on another shard by proving that the transaction was locked on the beacon chain.

## Beacon Chain

The beacon chain consists of validator registrations, attestations, withdrawals, and slashing proofs. The state of the beacon chain consists of active state which is updated every block and crystallized state which is updated every epoch. Epochs occur roughly every 100 blocks.

### Attestations

Attestations are votes by validators to finalize a shard block and finalize an epoch. According to the CASPER specification, votes consist of a source epoch and a destination epoch. Attestation votes use `EpochBoundaryHash` as the source epoch and `JustifiedSlot`/ `JustifiedBlockHash` as the destination epoch.

### Shard/Block Assignments

At the start of each cycle, validators are shuffled into different committees. Each committee is assigned to a certain shard and needs to sync up with that shard quickly in order to attest to shard blocks.

Each committee consists of a proposer and at least 128 attesters. Committees attest to a certain shard at a certain slot. Those attestations are then submitted to the mempool individually. After verifying the attestations, and waiting a certain amount of time after the attestation took place (`MinAttestationInclusionDelay`), proposers aggregate all of the attestations and include them in a block.

The crystallized state keeps track of a list of slots, each with a list of committees assigned to that slot. Each committee is also assigned a `ShardID` and a list of validator indices. Validators in the committee can be looked up by plugging the validator ID into `State.CrystallizedState.Validators\[validatorID\]`. The `slot % len(committee)` validator in the first committee of each slot is assigned to submit the beacon chain block.

### Entropy

To prevent an attacker from controlling more than 2/3 of the validators on a committee, the committees need to be securely randomly shuffled. This is accomplished using a RANDAO.

1. Every block, validators must publish the signature: `sha256(slotNumber)` with the public key of the proposer.
2. The `RandaoMix` from the previous epoch is XOR'd with each block in the chain to form the next `RandaoMix`.
3. If the validator does not reveal their RANDAO secret, they lose the opportunity to create a block.

This way, the only control over the `RANDAOMix` a validator has is whether to propose a block or not. If the validator does not propose a block, they lose out on the rewards for the block.

## Slashings

Slashings occur when a validator acts maliciously. Any of these actions will cause the validator to be moved into status `ExitedWithPenalty` and exited as soon as possible. The following actions can cause a slashing:

- CASPER
  - Attesting to beacon epochs surrounded by each other.
    - ex. Voting 1 - 4 and then voting 2 - 3 or Voting 1 - 3
  - Attesting to different shard hashes at the same height
    - ex. Voting epoch 1 with hash: `abc` and voting epoch 1 with hash: `def`.
- Proposing a block at the same height with a different hash

Violating any of these causes the validator to be immediately exited (moved to state `ExitedWithPenalty`) and slashed `1/whistleblowerRewardQuotient` of their deposit which is then given to the proposer who submitted the slashing.

## Rewards

For all active validators, rewards of `balance(index) / baseRewardQuotient / 5 \* previousEpochHeadAttestingBalance / totalBalance` are given:

- if the validator submitted an attestation that attested to the correct beacon hash (attestations where `att.Data.BeaconBlockHash == HashAtSlot(att.Data.Slot)`)
- if the validator attested to the correct epoch boundary `EpochBoundaryHash == HashAtSlot(slot - epochLength \* 2)`
- if the validator attested to the correct justified slot `a.Data.JustifiedSlot == s.PreviousJustifiedSlot`

If a validator does not do those things, the validator loses part of their deposit.

After not finalizing an epoch for 4 epochs, any active validators lose `balance(index) / baseRewardQuotient` per epoch and any active validators who attest lost `inactivityPenalty(index)`

The proposer who included each attestation gains `baseReward(index) / IncluderRewardQuotient`

All attesters are rewarded `baseReward(index) \* MinInclusionDelay / att.InclusionDistance` to incentivize attestations to be submitted sooner.

Voters for a winning crosslink gain `baseReward(index) \* totalAttestingBalance / totalBalance` and voters against a winning crosslink lose `baseReward(index).`

Validators under a minimum balance are exited (`ExitedWithoutPenalty`).
