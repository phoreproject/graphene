# Block Transition

This document will explain how block transitions work.

Block transitions occur only when a proposer submits a block, after the slot transition for the block's slot.

## Table of Contents

1. [Data Structure](#data-structure)
    1. [Block Header](#block-header)
    2. [Block Body](#block-body)
2. [Processing](#processing)

## Data Structure

Blocks consist of a `BlockHeader`, where metadata is stored and a `BlockBody` where transactions are stored.

### Block Header

Block header stores important information about the chain, state, and authentication from the validator.

- `SlotNumber` - slot number of the block; the slot transition for this slot should be processed before the block transition
- `ParentRoot` - hash of the previous block
- `StateRoot` - merkle root of the state
- `RandaoReveal` - signature signing the slot of the block
- `Signature` - signature signing the block

### Block Body

The block body stores a certain number of 5 types of important transactions:

- `Attestations` - votes for CASPER and shard blocks
- `ProposerSlashing` - submitted when a validator breaks a proposal rule (i.e. proposing two blocks at the same time)
- `CasperSlashing` - submitted when a validator breaks a CASPER rule (i.e. double vote, surround vote, etc)
- `Deposit` - starts the process of entering a new validator
- `Exit` - starts the process of exiting an existing validator

## Processing

The proposer of a block is defined by the following algorithm:

```python
state_slot = s.epoch_index * epoch_length
slot_index = block.slot - 1 - state_slot + state_slot % epoch_length + epoch_length
first_commitee = s.shard_committees_at_slots[slot_index][0].Committee
proposer_index = first_committee[(slot-1) % len(first_committee)]
```

The `RandaoReveal` property of the block must verify with the proposer's public key, the hash of the slot number, and the domain `DomainRandao`.

The node should calculate the proposal root by calculating the hash of:

```python
block_without_signature = block.copy()
block_without_signature.header.signature = bls.EmptySignature

proposal = ProposalSignedData(
	slot=block.Slot,
    shard=beacon_shard_number,
    block_hash=hash(block_without_signature)
)
```

Then, the node should validate that the signature in the block validates with the proposer's public key, the hash of the `ProposalSignedData` and the domain `DomainProposal`.

Update `state.randao_mix` by XORing it with the hash of the RANDAO signature:  `new_mix = old_mix ^ hash(block.randao_reveal)`.

Ensure that `Attestation`, `CasperSlashing`, `ProposerSlashing`, `Deposit`, and `Exit` objects do not exceed the maximum allowed as specified in the config.
