package gid

import (
	"bytes"
	"runtime"
	"strconv"

	"github.com/rs/zerolog"
)

func Get() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

type ZLogHook struct{}

func (h ZLogHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	e.Uint64("gr", Get())
}
