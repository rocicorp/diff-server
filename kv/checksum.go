package kv

import (
	"fmt"
	"hash/crc32"
)

// Checksum represents a fast, incrementally computable,
// non-cryptographic checksum of the contents of a kv store.
type Checksum struct {
	value uint32
}

// Value returns the current checksum value.
func (c Checksum) Value() string {
	return fmt.Sprintf("%08x", c.value)
}

func compute(key []byte, value string) uint32 {
	keyBytes := append([]byte("K="), key...)
	keyLen := []byte(fmt.Sprintf("KL=%d", len(key)))
	valBytes := append([]byte("V="), []byte(value)...)
	valLen := []byte(fmt.Sprintf("VL=%d", len(value)))
	totalLen := len(keyLen) + len(keyBytes) + len(valLen) + len(valBytes)
	input := make([]byte, totalLen)
	var i int
	i += copy(input[i:], keyLen)
	i += copy(input[i:], keyBytes)
	i += copy(input[i:], valLen)
	copy(input[i:], valBytes)
	return crc32.ChecksumIEEE(input)
}

// Add adds an entry to the checksum.
func (c *Checksum) Add(key []byte, value string) {
	c.value ^= compute(key, value)
}

// Remove removes an entry from the checksum.
func (c *Checksum) Remove(key []byte, value string) {
	c.value ^= compute(key, value)
}

// Replace replaces a key's value in the checksum.
func (c *Checksum) Replace(key []byte, oldValue, newValue string) {
	c.Remove(key, oldValue)
	c.Add(key, newValue)
}

// Equal returns true if two checksums are equal.
func (c Checksum) Equal(c2 Checksum) bool {
	return c.Value() == c2.Value()
}
