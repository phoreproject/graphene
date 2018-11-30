// Package shard TODO: move it to proper package and folder
package shard

// HasBitSetAt return true if the bit at index is 1 in bitField
func HasBitSetAt(bitField []byte, index uint32) bool {
	byteIndex := index >> 3
	if byteIndex >= uint32(len(bitField)) {
		return false
	}

	return (bitField[byteIndex] & (0x80 >> (index % 8))) != 0
}
