package kv

import (
	"fmt"
	"hash/crc32"
	"strconv"
)

// Checksum represents a fast, incrementally computable,
// non-cryptographic checksum of the contents of a kv store.
type Checksum struct {
	value uint32
}

func (c Checksum) String() string {
	return fmt.Sprintf("%08x", c.value)
}

// FromString parses a checksum value from a string.
func (c *Checksum) FromString(s string) error {
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return err
	}
	c.value = uint32(v)
	return nil
}

func hashEntry(key string, value []byte) uint32 {
	keyLen := []byte(fmt.Sprintf("%d", len(key)))
	valLen := []byte(fmt.Sprintf("%d", len(value)))
	keyBytes := []byte(key)
	totalLen := len(keyLen) + len(keyBytes) + len(valLen) + len(value)
	input := make([]byte, totalLen)
	var i int
	i += copy(input[i:], keyLen)
	i += copy(input[i:], keyBytes)
	i += copy(input[i:], valLen)
	copy(input[i:], value)
	// Note: we could probably avoid the above copies using crc32.Update.
	return crc32.ChecksumIEEE(input)
}

// Add adds an entry to the checksum.
func (c *Checksum) Add(key string, value []byte) {
	c.value ^= hashEntry(key, value)
}

// Remove removes an entry from the checksum.
func (c *Checksum) Remove(key string, value []byte) {
	c.value ^= hashEntry(key, value)
}

// Replace replaces a key's value in the checksum.
func (c *Checksum) Replace(key string, oldValue, newValue []byte) {
	c.Remove(key, oldValue)
	c.Add(key, newValue)
}

// Equal returns true if two checksums are equal.
func (c Checksum) Equal(c2 Checksum) bool {
	return c.value == c2.value
}

// Reset resets the checksum to zero.
func (c *Checksum) Reset() {
	c.value = 0
}
