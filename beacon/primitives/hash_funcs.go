package primitives

import "github.com/btcsuite/btcd/chaincfg/chainhash"

// Hashable is a type that can be hashed.
type Hashable interface {
	Hash() chainhash.Hash
}

// MerkleRootHash calculates the merkle root of a list of hashes.
func MerkleRootHash(hs []chainhash.Hash) chainhash.Hash {
	if len(hs) == 1 {
		return hs[0]
	}
	if len(hs)%2 == 0 {
		reducedHashes := make([]chainhash.Hash, len(hs)/2)
		for i := 0; i < len(hs)/2; i++ {
			reducedHashes[i] = chainhash.HashH(append(hs[i*2][:], hs[i*2+1][:]...))
		}
		return MerkleRootHash(reducedHashes)
	}
	reducedHashes := make([]chainhash.Hash, len(hs)/2+1)
	for i := 0; i < len(hs)/2; i++ {
		reducedHashes[i] = chainhash.HashH(append(hs[i*2][:], hs[i*2+1][:]...))
	}
	reducedHashes[len(hs)/2] = hs[len(hs)]
	return MerkleRootHash(reducedHashes)
}

// MerkleRoot finds the hash of a slice of hashables.
func MerkleRoot(h []Hashable) chainhash.Hash {
	hashes := make([]chainhash.Hash, len(h))
	for i := range h {
		hashes[i] = h[i].Hash()
	}
	return MerkleRootHash(hashes)
}
