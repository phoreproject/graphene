# P2P Specification

This document explains how P2P messages are sent and handled.

## Beacon v1.0.0 (on /phore/beacon/{version})
- synchronize blocks
    - local: send a GetBlockMessage with the correct locator hashes and stop hash
    - remote: send 1000 blocks in BlockMessage
    - repeat until HashStop matches LatestBlockHash
- ban if:
    - sending too many blocks
    - asking for block with different genesis hash
    - sending invalid block
    - sending invalid attestations
    - asking for attestations more than once every 5 minutes

## Shard v1.0.0 (on /phore/shard/{shardnumber}/{version})
- synchronize blocks
    - local: send a GetShardBlocksMessage with the correct locator hashes and stop hash
    - remote: send a ShardBlockMessage with shard blocks requested
    - repeated until HashStop matches LatestBlockHash
- ban if:
    - too many blocks
    - invalid blocks
    - asking for block with wrong genesis hash

## Relayer v1.0.0 (on /phore/relayer/{shardnumber}/{version})
- synchronize packages
    - local: send a GetPackagesMessage to all advertising peers
    - remote: send a PackageMessage
- ban if:
    - asking for package more than once every slot
    - sending invalid package
