package repm

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aboodman/replicant/util/time"
)

func mm(assert *assert.Assertions, in interface{}) []byte {
	r, err := json.Marshal(in)
	assert.NoError(err)
	return r
}

func TestBasic(t *testing.T) {
	defer time.SetFake()()
	defer fakeUUID()()

	assert := assert.New(t)
	dir, err := ioutil.TempDir("", "")
	Init(dir, "")
	res, err := Dispatch("db1", "open", nil)
	assert.Nil(res)
	assert.NoError(err)
	resp, err := Dispatch("db1", "put", []byte(`{"id": "foo", "value": "bar"}`))
	assert.Equal(`{"root":"3aktuu35stgss7djb5famn6u7iul32nv"}`, string(resp))
	assert.NoError(err)
	resp, err = Dispatch("db1", "get", []byte(`{"id": "foo"}`))
	assert.Equal(`{"has":true,"value":"bar"}`, string(resp))
	resp, err = Dispatch("db1", "del", []byte(`{"id": "foo"}`))
	assert.Equal(`{"ok":true,"root":"d3dqs6rctj3bmqg43pctpe9jol01h5kl"}`, string(resp))
	testFile, err := ioutil.TempFile(connections["db1"].dir, "")
	assert.NoError(err)

	resp, err = Dispatch("db2", "put", []byte(`{"id": "foo", "value": "bar"}`))
	assert.Nil(resp)
	assert.EqualError(err, "specified database is not open")

	resp, err = Dispatch("db1", "close", nil)
	assert.Nil(resp)
	assert.NoError(err)

	resp, err = Dispatch("db1", "put", []byte(`{"id": "foo", "value": "bar"}`))
	assert.Nil(resp)
	assert.EqualError(err, "specified database is not open")

	resp, err = Dispatch("db1", "drop", nil)
	assert.Nil(resp)
	assert.NoError(err)
	fi, err := os.Stat(testFile.Name())
	assert.Equal(nil, fi)
	assert.True(os.IsNotExist(err))
}
