# Block Processing

Block processing happens whenever a new block is received. This includes validation, state transition, block index
update, state storage, etc.

## Terminology

**State Map** - maps block hashes to the state after processing them

**Block Database** - database mapping block hashes to full blocks

**Block Index** - in-memory representation of the blockchain including all blocks

## Important Types

- Block
  - full block including attestations/exits/deposits/slashings
- State
  - full state of blockchain (equivalent to UTXO set in BTC)
- Block Node
  - Block hash
- Block height
  - Slot
  - Parent
  - Children

---
Side note: block `height != slot` because slots progress even if blocks aren't proposed.

|Slot|Height|
|---|---|
|0|0
|1|n/a|
|2|n/a|
|3|1|
|4|2|

## Storage

- Database
- Blocks
- Finalized/Justified Blocks
- Block Index
- Latest Attestations from each Validator
- Genesis Time
- Map of block hash to update state
- State Map
- Map of block hash to updated state
- Block Index
- Finalized block node and state
- Justified block node and state
- Tip block node
- Map of block hash to block node

## State Transition

State transitions are done in 3 phases.

- **Slot Transition**
- happen every single slot regardless of whether a block was proposed
- **Block Transition**
- happen after the slot transition for `block.slot`
- only happen if a block was proposed
- **Epoch Transition**
- happen regardless of whether a block was proposed
- run at the end of `state.slot` if `state.slot % epochLength == 0`

## Process

1. Validation
  a. Make sure the block isn't processed too soon. A block can only be created on or after `genesisTime + slotNumber * seconds per slot`.
  b. Ensure the block has a parent block in the block index.
  c. Ensure we haven't already processed a block with this hash.
2. Calculate block state and add to state map. (`blockchain.StateManager.AddBlockToStateMap`)
  a. State update phase
    i. Run slot transitions until `state.slot == block.slot`
    ii. Run block transition
    iii. If `slot % epochLength == 0`, run epoch transition
  b. Set the block hash in the database to the updated state
  c. Set the block hash in the memory state map to the updated state
3. Store block in database (`blockchain.StoreBlock`)
4. Add block to block index (`blockchain.addBlockNodeToIndex`)
5. Update database block index
6. Update latest attestations for each validator
7. Update chain tip
8. Update the finalized and justified nodes and corresponding states.
9. Delete any states that have `slot < finalizedNode.slot`
