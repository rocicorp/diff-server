package kv

import (
	"fmt"
	"hash/crc32"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChecksumComputeAndValue(t *testing.T) {
	assert := assert.New(t)

	var c Checksum
	assert.Equal("00000000", c.Value())

	// Might look like a dumb test but it caught two errors in the
	// original implementation.
	k := []byte{0x01, 0x02}
	v := "value⌘" // ⌘ is 3 bytes, ensuring code is counting bytes not runes
	expectedInput := []byte{
		0x32,       // '2'
		0x01, 0x02, // {0x01, 0x02}
		0x38,                                           // '8'
		0x76, 0x61, 0x6c, 0x75, 0x65, 0xe2, 0x8c, 0x98, // 'v''a''l'u''e''⌘'
	}
	c.Add(k, v)
	assert.Equal(fmt.Sprintf("%08x", crc32.ChecksumIEEE(expectedInput)), c.Value())
}

func TestChecksumOperations(t *testing.T) {
	assert := assert.New(t)

	k1, v1 := []byte{0x01}, "1"
	k2, v2 := []byte{0x02}, "2"
	var c1, c2 Checksum

	c1.Add(k1, v1)
	assert.True(c1.Equal(c1))
	assert.False(c1.Equal(c2))
	c2.Add(k2, v2)
	c2.Add(k1, v1)
	assert.False(c2.Equal(c1))
	c2.Remove(k2, v2)
	assert.True(c1.Equal(c2))

	c1.Replace(k1, v1, v2)
	var c3 Checksum
	c3.Add(k1, v2)
	assert.True(c3.Equal(c1))
}
