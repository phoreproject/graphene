# Mempool

This document describes the standard memory pool for the beacon chain which keeps track of: `Attestation`, `Exit`, `Deposit`, `CasperSlashing`, `ProposerSlashing` objects.

## Table of Contents

1. [Validation](#validation)
2. [Prioritization](#prioritization)
3. [Limits](#limits)

## Validation

TODO

## Prioritization

## Limits

Limits are imposed on blocks to prevent DoS attacks through block creation. For example, without limits, blocks could contains hundreds of separate attestations which would overload clients.

Currently, the limits per block are as follows:

|Object|Maximum Number Per Block|
|---|---|
|`ProposerSlashing`|16|
|`CasperSlashing`|16|
|`Attestation`|128|
|`Deposit`|16|
|`Exit`|16|
