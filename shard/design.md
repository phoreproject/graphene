# graphene Shard Design Document

## Overview

Each shard keeps a completely isolated state. Shards can run a few basic primitives provided by us.

- `broadcast(message: uint256)` - writes a broadcast message to the state
- `checkBroadcast(shard: uint32, message: uint256, proof: []uint256)` - checks that a certain shard has a certain broadcast
- `load(address: uint256)` - loads some memory from the specified address
- `save(address: uint256, value: uint256)` stores some memory to the specified address

There are two shards that are protected and can only be changed through a hard fork. The transfer shard allows users to send money and the governance shard allows users to vote for proposals and vote to update shard code.

## Transfer Shard

### Methods

- `Send(toAddress, shard, signature)`
  - if this is our shard, subtract from users balance and add to toAddress balance
  - if this is not our shard, subtract from users balance and broadcast `hash("transfer" + theirShard + address + nonce)` and increment nonce for this address

- `Receive(nonce, fromShard, proof)`
  - `checkBroadcast(fromShard, hash("transfer" + shard.id + from.address + nonce), proof)`
    - if no, exit
    - if yes,
      - mark broadcast as spent
      - increment balance of user on shard

## Governance Shard

### Overview

There are 3 types of proposals that can be voted on:

1. Budget Proposal (proposal to pay a certain address a certain amount per week for a certain number of weeks)
2. Budget Limit Proposal (proposal to raise/lower the budget to a certain amount per week)
3. Shard Code Proposal (proposal to replace/set the code for a specific shard to a certain hash of a wasm file)

### Methods

- `ProposeBudget(amountWeekly, numWeeks, toAddress, toShard, nonce) -> uint256`
  - proposes a budget `hash("proposal" + amountWeekly + numWeeks + toAddress + toShard + nonce)`
- `SetBudgetLimit(amountWeekly)`
  - proposes a vote to set the budget limit
- `SetShardCode(shardID, shardCodeHash)`
  - proposes a vote to set code for a specific shard
- `Vote(proposalID, validatorID, signature)`
  - votes for a certain proposal
- `RedeemBudget(toAddress, toShard, nonce)`
  - sends budget amount to redeem address if ready by broadcasting `hash("transfer" + toShard + toAddress + nonce)`
