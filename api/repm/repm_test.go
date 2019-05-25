package repm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelloWorld(t *testing.T) {
	assert := assert.New(t)

	conn, err := Open("/tmp/db1")
	assert.NoError(err)

	cmd, err := conn.Exec("data/put", []byte(`{"ID": "obj1"}`))
	assert.NoError(err)

	expected := `"Hello, Replicant"`
	n, err := cmd.Write([]byte(expected)[:5])
	assert.NoError(err)
	assert.Equal(5, n)
	n, err = cmd.Write([]byte(expected)[5:])
	assert.NoError(err)
	assert.Equal(len(expected)-5, n)

	_, err = cmd.Done()
	assert.NoError(err)

	cmd, err = conn.Exec("data/get", []byte(`{"ID": "obj1"}`))
	assert.NoError(err)

	buf := make([]byte, 5)
	n, err = cmd.Read(buf)
	assert.NoError(err)
	assert.Equal(5, n)
	buf = make([]byte, 1024)
	n, err = cmd.Read(buf)
	assert.NoError(err)
	assert.Equal(string(buf[:n]), expected[5:]+"\n")
}
