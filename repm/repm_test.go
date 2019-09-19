package repm

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aboodman/replicant/util/time"
)

func TestBasic(t *testing.T) {
	defer time.SetFake()()

	assert := assert.New(t)
	dir, err := ioutil.TempDir("", "")
	assert.NoError(err)
	conn, err := Open(dir, "test", "")
	assert.NoError(err)
	resp, err := conn.Dispatch("put", []byte(`{"id": "foo", "value": "bar"}`))
	assert.Equal(`{"root":"3aktuu35stgss7djb5famn6u7iul32nv"}`, string(resp))
	assert.NoError(err)
	resp, err = conn.Dispatch("get", []byte(`{"id": "foo"}`))
	assert.Equal(`{"has":true,"value":"bar"}`, string(resp))
	resp, err = conn.Dispatch("del", []byte(`{"id": "foo"}`))
	assert.Equal(`{"ok":true,"root":"d3dqs6rctj3bmqg43pctpe9jol01h5kl"}`, string(resp))
	testFile, err := ioutil.TempFile(dir, "")
	assert.NoError(err)

	resp, err = conn.Dispatch("dropDatabase", nil)
	assert.Nil(resp)
	assert.NoError(err)
	fi, err := os.Stat(testFile.Name())
	assert.Equal(nil, fi)
	assert.True(os.IsNotExist(err))
}
