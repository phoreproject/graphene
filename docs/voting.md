# Voting

This document will describe the voting process for the beacon chain. This governance installs new types of shards and upgrades existing shards for new governance

## Table of Contents

1. [Constants](#constants)
2. [Voting Process](#voting-process)
3. [Data Structures](#data-structures)
4. [State Processing](#state-processing)

## Constants

| Constant Name      | Value               |
| ------------------ | ------------------- |
| `VOTING_PERIOD`    | 14 (days)           |
| `QUEUE_THRESHOLD`  | 0.75                |
| `FAIL_THRESHOLD`   | 0.40                |
| `GRACE_PERIOD`     | 2 * `VOTING_PERIOD` |
| `CANCEL_THRESHOLD` | 0.5                 |
| `VOTING_TIMEOUT`   | 3 * `VOTING_PERIOD` |
| `PROPOSAL_COST`    | 1 PHR               |



## Voting Process

### Step 1: Get Proposal Queued

A proposal being queued does not mean that it will be implemented, but, unless validators change their mind, it will be implemented soon after.

1. A validator proposes a vote to implement new code on a number of shards.
2. Validators vote on the proposal.
3. If the vote gains support of `QUEUE_THRESHOLD` of validators, then the proposal is queued to be implemented.
4. If the vote gains support of `FAIL_THRESHOLD` of validators, but does not pass, then the proposal is added to the proposal list for the next voting period as long as the time since proposal is not more than `VOTING_TIMEOUT`.
5. If the vote does not gain support of `FAIL_THRESHOLD` of validators, the proposal fails.

### Step 2: Grace Period

After a proposal is queued, it will be implemented as long as validators do not cancel it.

To cancel it, a validator can submit a cancel proposal. If the cancel proposal gains more than `CANCEL_THRESHOLD` support for canceling the proposal, then the proposal will not be implemented and the vote is cancelled.

Note that in the simplest case where exactly 75% of validators vote for the proposal, cancellation requires that at least 1/3 (25% of the total) of them vote for cancelling the proposal.

## Data Structures

A `VoteData` represents which shards are being changed with what code.

```go
type VoteData struct {
    Type uint8 // 0: propose, 1: cancel
    Shards []uint8 // bitfield with the shards to implement the code in
    ActionHash chainhash.Hash // hash of code to implement or hash of VoteData for cancellation
    Proposer uint32 // proposer of the vote
}
```

- The n-th bit of `Shards` is 1 if the vote is proposing to implement the code hash on that shard.
- `CodeHash` is the hash of the WebAssembly code to be implemented.
- `Proposer` is the validator ID of the proposer.

`AggregatedVote` represents a vote with signatures from each validator that supports it. `AggregateVote` costs 1 PHR to propose if it has never been proposed before. Otherwise, it is free.

```go
type AggregatedVote struct {
    Data VoteData
    Signature bls.Signature
    Participation []uint8 // bitfield of validator IDs who support the proposal
}
```

`ActiveProposal` represents a proposal being voted on in state processing.

```go
type ActiveProposal struct {
    Data VoteData
    VoteParticipation []uint8 // 1 if the validator at index i voted for the proposal
    StartEpoch uint64
    Queued bool
}
```

`ShardRegistry` is a list of `ShardEntry` objects such that the i-th item corresponds to the i-th shard:

```go
type ShardEntry struct {
    CodeHash chainhash.Hash
}
```



## State Processing

First calculate `EpochsPerPeriod = VOTING_PERIOD * 24 * 3600 / slot_duration / epoch_duration`.

The following steps are run at the end of an epoch transition if `epoch_index % epochs_per_period == 0`.

1. Go through the `PendingVote` objects in order.
   1. If the vote is in `ActiveProposals`:
      1. If the vote is queued, ignore pending vote.
      2. Verify that the vote is valid by checking that the validators in the participation bitfield signed the `VoteData` with their public keys and domain `DomainGovernanceVote`.
      3. OR the active proposal participation with the participation bitfield from the vote.
   2. If the vote is not in `ActiveProposals`:
      1. If `vote.Data.Type == CANCEL`, ensure there is an `ActiveProposal` such that `hash(proposal.Data) == vote.Data.ActionHash`.
      2. Subtract `PROPOSAL_COST` from the validator's balance.
      3. Verify that the vote is valid by checking that the validators in the participation bitfield signed the `VoteData` with their public keys and domain `DomainGovernanceVote`.
      4. Create a new `ActiveProposal` with the `VoteData` specified and `VoteParticipation` such that the bit corresponding to the proposer is 1. Set the `StartEpoch` to the current epoch.
2. Go through each `ActiveProposal`:
   1. If not queued, calculate the Hamming weight of the VoteParticipation bitfield `number_yes_votes`.
      1. Calculate `voting_percent = number_yes_votes / len(s.ValidatorRegistry)`.
      2. If `proposal.Data.Type == PROPOSE`
         1. If `voting_percent > QUEUE_THRESHOLD`:
            1. Otherwise, set `Queued` to `true`.
      3. If `proposal.Data.Type == CANCEL`
         1. If `voting_percent > CANCEL_THRESHOLD`:
            1. Remove any proposals from `ActiveProposals` where `Queued` is true and `hash(proposal.Data) == queued_proposal.ActionHash`.
      4. If `voting_percent <= FAIL_THRESHOLD && proposal.Data.Type == PROPOSE`:
         1. Remove from `ActiveProposals`.
      5. If `EpochIndex - proposal.StartEpoch / EpochsPerPeriod > VOTING_TIMEOUT`:
         1. Remove from `ActiveProposals`.
   2. If queued,
      1. Check if `EpochIndex - proposal.StartEpoch / EpochsPerPeriod > GRACE_PERIOD`, then loop through each bit of `proposal.Data.Shards`:
         1. If bit `i` is a 1, set the `CodeHash` of `ShardRegistry[i]` to `proposal.Data.CodeHash`.
      2. Remove any proposals in ActiveProposals where `conflicting_proposal.Data.Type == CANCEL` and `conflicting_proposal.Data.ActionHash == hash(proposal.Data)`.
