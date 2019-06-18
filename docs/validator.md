# Validator

A validator has two main responsibilities: attesting to shard blocks to form cross-links and proposing beacon blocks by aggregating all of the attestations together and putting them into a block.

## Table of Contents

1. [Attestations](#attestations)
2. [Proposals](#proposals)

## Attestations

Attestations serve two main purposes in Phore Synapse. They:

a. act as a vote for beacon chain CASPER
b. act as a vote for a shard block

Attestations should be submitted halfway through a slot, or at `0.5 * slot_duration + slot_number * slot_duration + genesis_time`.

At this point, the validator should call, `UpdateEpochInformation(slot_number / epoch_length)` to retrieve the current epoch information from the beacon chain. The current epoch information includes:

```go
type epochInformation struct {
  slots                  []slotInformation
  slot                   int64
  targetHash             chainhash.Hash
  justifiedEpoch         uint64
  latestCrosslinks       []primitives.Crosslink
  previousCrosslinks     []primitives.Crosslink
  justifiedHash          chainhash.Hash
  epochIndex             uint64
  previousTargetHash     chainhash.Hash
  previousJustifiedEpoch uint64
  previousJustifiedHash  chainhash.Hash
}
```

The validator can then access information about current epoch assignments through the `Slots` field, which includes information about the proposal time, the committees and the slot number of each slot in the array.

The index into the `Slots` field can be calculated as follows:

```python3
epoch_offset = current_slot - vm.latestEpochInformation.slot - 1
```

The committees for the slot can are stored in `Slots[epoch_offset]`.

Also, find the beacon block hash corresponding to the `current_slot` by calling `GetBlockHash` with `current_slot` and obtaining `current_slot_hash`.

For each of those committees, run the following for every validator in the committee that is being controlled by the validator module:

### Single validator attesting to a single committee

Let `committee` be the committee of the validator.

Calculate the `data` field using the following values:

```python3
data = AttestationData(
  slot=current_slot,
  shard=committee.shard,
  beacon_block_hash=current_slot_hash,
  latest_crosslink_hash=latest_crosslinks[committee.shard].hash if current_slot%epoch_length != 0 else previous_crosslinks[committee.shard].hash,
  source_epoch=justified_epoch if current_slot%epoch_length != 0 else previous_justified_epoch,
  source_hash=justified_hash if current_slot%epoch_length != 0 else previous_justified_hash,
  target_epoch=epoch_index if current_slot%epoch_length != 0 else epoch_index-1,
  target_hash=target_hash if current_slot%epoch_length != 0 else previous_target_hash,
  shard_block_hash=[0] * 32
)
```

Find `attestation_hash = Hash(AttestationDataAndCustodyBit(data=data, bit=False))` and sign the value with the validator's key and the domain: `version << 32 + DomainAttestation`. Store the signature in `signature`.

Set `committee_index` to the index in the committee of the validator and `committee_size` to the number of validators in the committee.

Then, calculate `participation_bitfield` by running the following code (1 << `committee_index`):

```python3
bitfield = [0] * (committee_size+7) // 8
bitfield[committee_index // 8] = 1 << (committee_index % 8)
```

Finally, submit the attestation with the following values:

```python3
att = Attestation(
  data=data,
  participation_bitfield=participation_bitfield,
  custody_bitfield=[0] * (committee_size+7)//8,
  aggregate_signature=signature,
)
```

## Proposals

Proposals are beacon chain blocks where attestations, deposits, exits, and slashings are aggregated and packaged into a block, then signed by a proposer.

TODO...
