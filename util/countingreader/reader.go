package countingreader

import "io"

type Callback func()

type Reader struct {
	R        io.Reader
	Callback Callback
	Count    uint64
}

func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.R.Read(p)
	r.Count += uint64(n)
	r.Callback()
	return n, err
}
