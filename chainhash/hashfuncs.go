// Copyright (c) 2015 The Decred developers
// Copyright (c) 2016-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chainhash

import (
	"errors"
	"golang.org/x/crypto/blake2b"
)

// HashB calculates hash(b) and returns the resulting bytes.
func HashB(b []byte) []byte {
	hash := blake2b.Sum256(b)
	return hash[:]
}

// HashH calculates hash(b) and returns the resulting bytes as a Hash.
func HashH(b []byte) Hash {
	return Hash(blake2b.Sum256(b))
}

// HashesToBytes converts an array of hashes to byte arrays.
func HashesToBytes(h []Hash) [][]byte {
	out := make([][]byte, len(h))
	for i := range h {
		out[i] = h[i][:]
	}
	return out
}

// BytesToHash converts a byte array to a hash.
func BytesToHash(b []byte) (Hash, error) {
	if len(b) != 32 {
		return Hash{}, errors.New("expected hash to be length 32")
	}
	var out Hash
	copy(out[:], b)
	return out, nil
}

// BytesToHashes converts an array of byte arrays to hashes.
func BytesToHashes(b [][]byte) ([]Hash, error) {
	out := make([]Hash, len(b))

	for i := range out {
		h, err := BytesToHash(b[i])
		if err != nil {
			return nil, err
		}
		out[i] = h
	}

	return out, nil
}