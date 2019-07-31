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
			"bundle put good",
			"function futz(k, v){ db.put(k, v) }",
			"bundle put",
			0,
			"",
			"",
		},
		{
			"bundle get good",
			"",
			"bundle get",
			0,
			"function futz(k, v){ db.put(k, v) }",
			"",
		},
		{
			"exec unknown-function",
			"",
			"exec monkey",
			1,
			"",
			"Error: Unknown function: monkey\n",
		},
		{
			"exec missing-key",
			"",
			"exec futz",
			1,
			"",
			"Error: Invalid id\n",
		},
		{
			"exec missing-val",
			"",
			"exec futz foo",
			1,
			"",
			"Error: Invalid value\n",
		},
		{
			"exec good",
			"",
			"exec futz foo bar",
			0,
			"",
			"",
		},
		{
			"has missing-arg",
			"",
			"has",
			1,
			"",
			"required argument 'id' not provided\n",
		},
		{
			"has good",
			"",
			"has foo",
			0,
			"true\n",
			"",
		},
		{
			"get bad missing-arg",
			"",
			"get",
			1,
			"",
			"required argument 'id' not provided\n",
		},
		{
			"get good",
			"",
			"get foo",
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
