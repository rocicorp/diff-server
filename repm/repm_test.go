package repm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	assert := assert.New(t)
	conn, err := Open("mem", "test")
	assert.NoError(err)
	resp, err := conn.Dispatch("put", []byte(`{"key": "foo", "data": "bar"}`))
	assert.Equal([]byte(`{}`), resp)
	assert.NoError(err)
	resp, err = conn.Dispatch("get", []byte(`{"key": "foo"}`))
	assert.Equal([]byte(`{"has":true,"data":"bar"}`), resp)
}
