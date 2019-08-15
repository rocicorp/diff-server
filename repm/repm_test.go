package repm

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	assert := assert.New(t)
	dir, err := ioutil.TempDir("", "")
	assert.NoError(err)
	conn, err := Open(dir, "test", "")
	assert.NoError(err)
	resp, err := conn.Dispatch("put", []byte(`{"key": "foo", "data": "bar"}`))
	assert.Equal([]byte(`{}`), resp)
	assert.NoError(err)
	resp, err = conn.Dispatch("get", []byte(`{"key": "foo"}`))
	assert.Equal([]byte(`{"has":true,"data":"bar"}`), resp)
	resp, err = conn.Dispatch("del", []byte(`{"key": "foo"}`))
	assert.Equal([]byte(`{"ok":true}`), resp)
	testFile, err := ioutil.TempFile(dir, "")
	assert.NoError(err)

	resp, err = conn.Dispatch("dropDatabase", nil)
	assert.Nil(resp)
	assert.NoError(err)
	fi, err := os.Stat(testFile.Name())
	assert.Equal(nil, fi)
	assert.True(os.IsNotExist(err))
}
