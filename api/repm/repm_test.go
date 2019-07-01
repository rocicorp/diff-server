package repm

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelloWorld(t *testing.T) {
	assert := assert.New(t)

	tc := []struct {
		label string
		cmd   string
		in    string
		args  string
		res   string
		out   string
		err   string
	}{
		{
			"code/put good",
			"code/put",
			"function futz(k, v){ db.put(k, v) }",
			"",
			"",
			"",
			"",
		},
		{
			"code/get good",
			"code/get",
			"",
			"",
			`{"OK":true}`,
			"function futz(k, v){ db.put(k, v) }",
			"",
		},
		{
			"code/run missing-func",
			"code/run",
			"",
			`{}`,
			"",
			"",
			"Function name parameter is required",
		},
		{
			"code/run unknown-function",
			"code/run",
			"",
			`{"Name": "monkey"}`,
			"",
			"",
			"Error: Unknown function: monkey",
		},
		{
			"code/run missing-key",
			"code/run",
			"",
			`{"Name": "futz"}`,
			"",
			"",
			"Error: Invalid id",
		},
		{
			"code/run missing-val",
			"code/run",
			"",
			`{"Name": "futz", "Args": ["foo"]}`,
			"",
			"",
			"Error: Invalid value",
		},
		{
			"code/run good",
			"code/run",
			"",
			`{"Name": "futz", "Args": ["foo", "bar"]}`,
			"",
			"",
			"",
		},
		{
			"data/has good",
			"data/has",
			"",
			`{"ID": "foo"}`,
			`{"OK":true}`,
			"",
			"",
		},
		{
			"data/get good",
			"data/get",
			"",
			`{"ID": "foo"}`,
			`{"OK":true}`,
			"\"bar\"\n",
			"",
		},
	}

	td, err := ioutil.TempDir("", "")
	assert.NoError(err)

	conn, err := Open(td)
	assert.NoError(err)

	for _, c := range tc {
		cmd, err := conn.Exec(c.cmd, []byte(c.args))
		assert.NotNil(cmd, c.label)
		assert.NoError(err, c.label)
		if c.in != "" {
			n, err := cmd.Write([]byte(c.in))
			assert.NoError(err, c.label)
			assert.Equal(len(c.in), n, c.label)
		}

		if c.out != "" {
			buf := &bytes.Buffer{}
			_, err := io.Copy(buf, cmd)
			assert.NoError(err, c.label)
			assert.Equal(len(c.out), buf.Len(), c.label)
			assert.Equal(c.out, string(buf.Bytes()), c.label)
		}

		res, err := cmd.Done()
		if c.err != "" {
			assert.Nil(res, c.label)
			assert.NotNil(err, c.label)
			assert.Equal(c.err, err.Error(), c.label)
		} else {
			assert.NoError(err, c.label)
			assert.Equal(c.res, string(res), c.label)
		}
	}
}
