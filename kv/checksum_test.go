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
	assert.Equal("00000000", c.String())

	// Might look like a dumb test but it caught two errors in the
	// original implementation.
	k := "key⌘" // ⌘ is 3 bytes, ensuring code is counting bytes not runes
	v := []byte{0x01, 0x02}
	expectedInput := []byte{
		0x36,                               // '6'
		0x6B, 0x65, 0x79, 0xe2, 0x8c, 0x98, // 'k''e''y''⌘'
		0x32,       // '2'
		0x01, 0x02, // {0x01, 0x02}
	}
	c.Add(k, v)
	assert.Equal(fmt.Sprintf("%08x", crc32.ChecksumIEEE(expectedInput)), c.String())
}

func TestChecksumOperations(t *testing.T) {
	assert := assert.New(t)

	k1, v1 := "1", []byte{0x01}
	k2, v2 := "2", []byte{0x02}
	var c1, c2 Checksum

	c1.Add(k1, v1)
	assert.True(c1.Equal(c1))
	assert.False(c1.Equal(c2))
	c1.Reset()
	assert.True(c1.Equal(c2))

	c1.Add(k1, v1)
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

func TestChecksum_FromString(t *testing.T) {
	type fields struct {
		value uint32
	}
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		value   uint32
		s       string
		wantVal uint32
		wantErr bool
	}{
		{"parses", 0, "00cf3d55", 13581653, false},
		{"error", 0, "p00p", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Checksum{
				value: tt.value,
			}
			if err := c.FromString(tt.s); (err != nil) != tt.wantErr {
				t.Errorf("Checksum.FromString() error = %v, wantErr %v", err, tt.wantErr)
			}
			if c.value != tt.wantVal {
				t.Errorf("Checksum.FromString() got = %v, want %v", c.value, tt.wantVal)
			}
		})
	}
}
