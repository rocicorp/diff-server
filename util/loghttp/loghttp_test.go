package loghttp

import (
	"bytes"
	"compress/gzip"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_maybeUnzip(t *testing.T) {
	assert := assert.New(t)

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write([]byte("this is gzipped"))
	assert.NoError(err)
	assert.NoError(zw.Close())

	tests := []struct {
		in   []byte
		want []byte
	}{
		{
			[]byte{},
			[]byte{},
		},
		{
			[]byte("\n"),
			[]byte("\n"),
		},
		{
			[]byte("not gzipped"),
			[]byte("not gzipped"),
		},
		{
			buf.Bytes(),
			[]byte("this is gzipped"),
		},
	}
	for i, tt := range tests {
		got, err := maybeUnzip(tt.in)
		assert.NoError(err)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%d: maybeUnzip() = %v, want %v", i, got, tt.want)
		}
	}
}
