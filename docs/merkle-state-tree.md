# Merkle State Tree

The merkle state tree stores the entire state of the Synapse shards. If a contract needs to lookup the value of a certain key in state, a witness proving the value of the key is included with the transaction so that executors can run the transaction without the entire state. Similarly, if a contract needs to update the value of a certain key in state, a witness allowing an executor to update the value of the key is included with the transaction. This "update witness" allows executors, using the previous state root, the calculate a new state root with only the single key changed to the new value.

## Structure

The merkle state tree is a sparse merkle tree. This means that every value in the state is already included and initially set to 0. The leaf nodes represent the hash of the value.

For example, all leaf nodes are initially set to `Hash(value) = Hash(0)`. Values in the tree are indexed based on the key. The first bit in the hash of the key represents whether the key is in the left or right subtree from the root (`0` if left and `1` if right). The second bit represents whether the key is in the left or right subtree from the level below the root, etc. This means that the far-left node is a key with hash `0000...000` and the far-right node is a key with hash `1111...111`.

To calculate the hash of the tree, we don't actually need to calculate 2^256^ different hash values. Instead, notice that all leaf nodes have hash `Hash(0)`, all parents of leaf nodes have hash `Hash(Hash(0) || Hash(0))`, etc. We can pre-compute each of the 256 different levels of empty sub-trees.

## Compressed Witness Format

Witnesses are especially important in our merkle state tree construction. Naively, witnesses would need to have `256 * 32 bytes = 8 kB` per witness, but most of these hashes will be empty subtrees. Thus, we can include a bitmap at the beginning to indicate which hashes are actually non-empty. Instead of `[h1, empty2, empty3, h4, empty5]`, we can instead use `(10010, [h1, h4])`. Thus, the space required for a single witness drops from `256 * 32` to `32 + 32 * lg(n)`. For 1 billion keys, a single witness would take `32 + 32 * lg(10^9) = 1 kB`. Further optimizations include batching updates in different subtrees and including each hash only once, so if a subtree isn't changed, we don't have to send it again.

### Calculate Root From Witness

To calculate the root from a witness,

```python
zeroHashes = [0] * 256

zeroHashes[0] = hash(0)

for i in range(1, 256):
    zeroHashes[i] = blake2(zeroHashes[i - 1] + zeroHashes[i - 1])

def calculate_root(key, value, witness_bitfield, witnesses):
    hk = blake2(key)
    root = hv = blake2(value)
    current_witness = 0
    for i in range(256):
        bit = witness_bitfield & (1 << i) != 0
        if bit:
            h = witnesses[current_witness]
            current_witness += 1
        else:
            h = zeroHashes[i]
        root = blake2(hv + h) if hk & (1 << (255-i)) == 0 else blake2(h + hv)
    return root

```



### Update Witness

For update witnesses, we want to allow the executor to generate a new state root such that no other key than the specified value is changed.

It includes the following fields:

- `key` - the key we want to change
- `prev_value` - the value the key has before the update
- `new_value` - the value the key will have after the update
- `witness_bitfield` - bit i is 1 if the i'th level (starting from the leaf) is non-empty on the merkle branch
- `witnesses` - every hash needed that isn't an empty sub-tree; `len(witnesses) == num_ones(witness_bitfield)`

First, we verify that the previous value and the key matches the current state root. Verify that `calculate_root(key, prev_value, witness_bitfield, witnesses) == current_state_root`.

Next, we should calculate the new state root with the value of the key changed. Return `calculate_root(key, new_value, witness_bitfield, witnesses)`.

### Verification Witness

For verification witnesses, we want to prove that a certain key is equal to a certain value. It includes the following fields:

- `key` - the key we want to check
- `value` - the value we want to check
- `witness_bitfield` - bit i is 1 if the i'th level (starting from the leaf) is non-empty on the merkle branch
- `witnesses` - every hash needed that isn't an empty sub-tree; `len(witnesses) == num_ones(witness_bitfield)`

Verify that `calculate_root(key, value, witness_bitfield, witnesses) == current_state_root`.
