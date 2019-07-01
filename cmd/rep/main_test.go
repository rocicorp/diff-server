package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommands(t *testing.T) {
	assert := assert.New(t)

	tc := []struct {
		label string
		in    string
		args  string
		code  int
		out   string
		err   string
	}{
		{
			"code put good",
			"function futz(k, v){ db.put(k, v) }",
			"code put",
			0,
			"",
			"",
		},
		{
			"code get good",
			"",
			"code get",
			0,
			"function futz(k, v){ db.put(k, v) }",
			"",
		},
		{
			"code run unknown-function",
			"",
			"code run monkey",
			1,
			"",
			"Error: Unknown function: monkey\n",
		},
		{
			"code run missing-key",
			"",
			"code run futz",
			1,
			"",
			"Error: Invalid id\n",
		},
		{
			"code run missing-val",
			"",
			"code run futz foo",
			1,
			"",
			"Error: Invalid value\n",
		},
		{
			"code run good",
			"",
			"code run futz foo bar",
			0,
			"",
			"",
		},
		{
			"data has missing-arg",
			"",
			"data has",
			1,
			"",
			"required argument 'ID' not provided\n",
		},
		{
			"data has good",
			"",
			"data has foo",
			0,
			"true\n",
			"",
		},
		{
			"data get bad missing-arg",
			"",
			"data get",
			1,
			"",
			"required argument 'ID' not provided\n",
		},
		{
			"data get good",
			"",
			"data get foo",
			0,
			"\"bar\"\n",
			"",
		},
	}

	td, err := ioutil.TempDir("", "")
	fmt.Println("test database:", td)
	assert.NoError(err)

	for _, c := range tc {
		ob := &strings.Builder{}
		eb := &strings.Builder{}
		code := 0
		args := append([]string{"--db=" + td}, strings.Split(c.args, " ")...)
		impl(args, strings.NewReader(c.in), ob, eb, func(c int) {
			code = c
		})

		assert.Equal(c.code, code, c.label)
		assert.Equal(c.out, ob.String(), c.label)
		assert.Equal(c.err, eb.String(), c.label)
	}

}
