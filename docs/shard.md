# Shard Module

The shard module is responsible for downloading state from peers, managing state storage, and executing state at the request of validators.

## Table of Contents

1. [Interactions](#interactions)
   1. [Beacon Module](#beacon-module)
   2. [Validator Module](#validator-module)
2. [Shard Design](#shard-design)
3. [State Download & Execution](#shard-download-and-execution)

## Interactions

The shard module contacts both the beacon module and the validator module to manage shard state execution.

### Beacon Module

The shard module contacts the beacon chain to:

- ask for the hash and ABI of the WebAssembly code to execute on the shard
- ask for information about current crosslinks (aka finalized shard states)

### Validator Module

The shard module mostly just handles requests from the validator module. The validator module can request the shard module to:

- predownload transactions/witnesses for some shards and start listening for blocks on these shards (there should be two possible shards to look ahead to: the shard where the validators are shuffled and the shard where the validators are not shuffled)
- given the old state root, calculate the new state root after applying the transactions specified
- submit a block (collation) to the shard
- stop listening to a specific shard

## Shard Design

Shards in Phore graphene work completely statelessly. All state is kept track of using transaction data. For example, assume Bob wants to transfer 10 PHR to Alice. Bob would need to submit the following pieces of information to form a valid transaction:

- The merkle proof of Bob's balance in the state tree
- The merkle proof of Alice's balance in the state tree
- Alice's public key
- Bob's signature of "send 10 PHR to Alice"
- The message "send 10 PHR to Alice"

In the beginning, state will be stored by altruistic validators, but will be incentivized eventually using custody bonds. This basically allows the network to pay validators for storing state for a certain period of time.

## Shard Download And Execution

The shard module must first download any blocks starting with the latest crosslink. The shard module will apply these blocks until it gets to the slot it should propose. Then, the shard module will apply any outstanding transactions, update the state root, submit the new shard block, and return the state root for processing.

The following communication happens between the validator and shard modules when proposing a block:

1. VM tells SM two possible shards to sync up for next epoch.
2. SM downloads and subscribes to new blocks and new transactions for both shards.
3. VM receives assignment from BM about which shard to attest and propose.
4. VM tells SM to unsubscribe from the incorrectly guessed shard.
5. VM asks SM for the crosslink hash at the previous boundary slot.
6. If VM is assigned to propose a shard block,
   1. At the assigned proposal time, VM asks SM to generate a block
   2. VM signs the block and submits it back to SM
   3. SM broadcasts the block to the network
7. Unsubscribe from the assigned shard.
8. Repeat every epoch.
