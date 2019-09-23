package log

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"runtime"
	"strconv"
)

type writer struct {
	opts Options
	w    io.Writer
}

func (w writer) Write(p []byte) (n int, err error) {
	if w.opts.Prefix {
		return io.WriteString(w.w, fmt.Sprintf("GR%09x %s",
			getGID(),
			p))
	}
	return w.w.Write(p)
}

type Options struct {
	Prefix bool
}

func Init(out io.Writer, opts Options) {
	var flags = 0
	if opts.Prefix {
		flags = log.Ldate | log.Lmicroseconds | log.Lshortfile
	}
	log.SetFlags(flags)
	log.SetOutput(writer{
		opts,
		out,
	})
}

func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
