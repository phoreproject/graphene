package shard

import (
	"bytes"
	"fmt"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/csmt"
	"github.com/phoreproject/synapse/utils"
)

// PartialTreeSerializer serialize PartialTree
type PartialTreeSerializer struct {
	version uint16
}

// NewPartialTreeSerializer creates a new PartialTreeSerializer
func NewPartialTreeSerializer(version uint16) *PartialTreeSerializer {
	return &PartialTreeSerializer{
		version: version,
	}
}

// Serialize serialize a PartialTree
func (pts *PartialTreeSerializer) Serialize(ptree *PartialTree) []byte {
	keyMap := map[chainhash.Hash]uint16{}
	hashList := make([]chainhash.Hash, 0)

	for _, wt := range ptree.witnesses {
		for _, e := range wt.proof.GetEntryList() {
			hash := *e.GetHash()
			_, found := keyMap[hash]
			if !found {
				hashList = append(hashList, hash)
				keyMap[hash] = uint16(len(hashList) - 1)
			}
		}
	}

	buffer := &bytes.Buffer{}
	writer := utils.NewWriter(buffer)

	writer.WriteUint16(pts.version)

	writer.WriteBytes(ptree.rootHash[:])

	// assume up to 65535/2 hashes in a PartialTree
	writer.WriteUint16(uint16(len(hashList)))
	for _, hash := range hashList {
		writer.WriteBytes(hash[:])
	}

	writer.WriteUint16(uint16(len(ptree.witnesses)))
	for _, wt := range ptree.witnesses {
		writer.WriteUint8(uint8(wt.operation))
		writer.WriteBytes(wt.key[:])

		entryList := wt.proof.GetEntryList()
		writer.WriteUint16(uint16(len(entryList)))
		for _, e := range wt.proof.GetEntryList() {
			hash := *e.GetHash()
			index, _ := keyMap[hash]
			if e.GetDirection() == csmt.DirRight {
				index |= 0x8000
			}
			writer.WriteUint16(index)
		}
	}

	return buffer.Bytes()
}

// Deserialize deserialize a PartialTree
func (pts *PartialTreeSerializer) Deserialize(data []byte) (*PartialTree, error) {
	r := bytes.NewReader(data)
	reader := utils.NewReader(r)

	version, err := reader.ReadUint16()
	if err != nil {
		return nil, err
	}
	if version > pts.version {
		return nil, fmt.Errorf("Unsupported partial tree version")
	}

	rootHashData := make([]byte, chainhash.HashSize)
	_, err = reader.ReadBytes(rootHashData)
	if err != nil {
		return nil, err
	}
	rootHash, err := chainhash.NewHash(rootHashData)
	if err != nil {
		return nil, err
	}

	hashCount, err := reader.ReadUint16()
	if err != nil {
		return nil, err
	}

	hashList := make([]chainhash.Hash, hashCount)
	for i := 0; i < int(hashCount); i++ {
		h := make([]byte, chainhash.HashSize)
		_, err := reader.ReadBytes(h)
		if err != nil {
			return nil, err
		}
		hashList[i].SetBytes(h)
	}

	witnessCount, err := reader.ReadUint16()
	if err != nil {
		return nil, err
	}

	witnesses := make([]Witness, witnessCount)
	for i := 0; i < int(witnessCount); i++ {
		operation, err := reader.ReadUint8()
		if err != nil {
			return nil, err
		}
		h := make([]byte, chainhash.HashSize)
		_, err = reader.ReadBytes(h)
		if err != nil {
			return nil, err
		}
		key, err := chainhash.NewHash(h)
		if err != nil {
			return nil, err
		}

		entryCount, err := reader.ReadUint16()
		if err != nil {
			return nil, err
		}

		entryList := make([]*csmt.MembershipProofEntry, entryCount)
		for k := 0; k < int(entryCount); k++ {
			index, err := reader.ReadUint16()
			if err != nil {
				return nil, err
			}
			direction := csmt.DirLeft
			if index&0x8000 != 0 {
				index &= 0x7fff
				direction = csmt.DirRight
			}

			entry := csmt.NewMembershipProofEntry(&hashList[index], direction)
			entryList[k] = &entry
		}

		witnesses[i], err = newWitness(int(operation), *key, csmt.NewMembershipProof(entryList))
		if err != nil {
			return nil, err
		}
	}

	ptree := NewPartialTree(*rootHash, witnesses)

	return &ptree, nil
}
