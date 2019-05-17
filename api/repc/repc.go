// package repc implements a low-level C API to Replicant
// suitable for embedding within iOS, Android, desktop
// software, etc.
package main

// compile like:
// go build -buildmode=c-archive -o repc.a repc.go

// #include <stdio.h>
import "C"

import (
	"fmt"
	"io"
	"reflect"
	"sync/atomic"
	"unsafe"

	"github.com/aboodman/replicant/cmd"
	"github.com/aboodman/replicant/db"
	"github.com/attic-labs/noms/go/spec"
)

type ConnectionID int32
type ExecID int32

type execInfo struct {
	in  io.WriteCloser
	out io.ReadCloser
	err chan error
}

var (
	connections = map[ConnectionID]db.DB{}
	execs       = map[ExecID]*execInfo{}
	nextID      = int32(0)
)

// Open creates a connection to Replicant. Changes made by a connection are
// cached in the connection and only flushed when the `commit` command is
// executed.
//
// Currently, no concurrency of any kind is supported in the API. Callers
// do not need to call on a single thread, but any concurrent calls - even
// on different connections will either corrupt the internal state of the
// API or give incorrect results.
//
// TODO: Support concurrency. Need to think about the API layer, but more
// importantly the Replicant/Noms layers.
//
//export Open
func Open(dbSpec *C.char, dbSpecLen C.int, connID *C.int) (errMsg *C.char) {
	sp, err := spec.ForDatabase(C.GoStringN(dbSpec, dbSpecLen))
	if err != nil {
		return C.CString(err.Error())
	}
	db, err := db.Load(sp)
	if err != nil {
		return C.CString(err.Error())
	}
	conn := ConnectionID(atomic.AddInt32(&nextID, 1))
	connections[conn] = db
	*connID = C.int(conn)
	return (*C.char)(C.NULL)
}

// Exec begins execution of a command. The `cmd` is a JSON-formatted message
// describing the command to execute. See `cmd.Commands` for details.
// To write data to the input stream of the command, call `ExecWrite`. To
// read data from the output stream, call `ExecRead`. To complete the
// execution and get the error code, if any, call `ExecDone`.
//export Exec
func Exec(conn ConnectionID, cs unsafe.Pointer, csLen C.int, execID *C.int) (errMsg *C.char) {
	db, ok := connections[conn]
	if !ok {
		return C.CString("invalid connection")
	}

	id := ExecID(atomic.AddInt32(&nextID, 1))
	outR, inW, ec, err := cmd.DispatchString(db, cbufToByteSlice(cs, csLen))
	if err != nil {
		return C.CString(err.Error())
	}
	info := &execInfo{
		in:  inW,
		out: outR,
		err: ec,
	}
	execs[id] = info

	*execID = C.int(id)
	return (*C.char)(C.NULL)
}

// ExecWrite writes data to the input stream of an execution started previously with
// `Exec`.
//export ExecWrite
func ExecWrite(id C.int, data unsafe.Pointer, dataLen C.int) (errMsg *C.char) {
	info, ok := execs[ExecID(id)]
	if !ok {
		return C.CString(fmt.Sprintf("Invalid execID: %d", id))
	}
	// TODO: The copy here sucks. Add a freeBuffer() function host is supposed to
	// implement and call in exec goroutine after passing string to command. If
	// command needs string to last beyond, it should copy.
	info.in.Write(C.GoBytes(data, dataLen))
	return (*C.char)(C.NULL)
}

// ExecRead reads data from the output stream of an execution started previously with
// `Exec`. When ExecRead has read all data, buf will be NULL and bufLen zero.
//export ExecRead
func ExecRead(id C.int, buf unsafe.Pointer, bufLen C.int, readLen *C.int) (errMsg *C.char) {
	info, ok := execs[ExecID(id)]
	if !ok {
		return C.CString(fmt.Sprintf("Invalid execID: %d", id))
	}
	n, err := info.out.Read(cbufToByteSlice(buf, bufLen))
	if err != nil {
		if err == io.EOF {
			return (*C.char)(C.NULL)
		}
		*readLen = C.int(0)
		return C.CString(err.Error())
	}
	*readLen = C.int(n)
	return (*C.char)(C.NULL)
}

// ExecDone completes a command execution and returns the relevant error, if any.
//export ExecDone
func ExecDone(id C.int) (errMsg *C.char) {
	info, ok := execs[ExecID(id)]
	if !ok {
		return C.CString(fmt.Sprintf("Invalid execID: %d", id))
	}
	delete(execs, ExecID(id))
	err := info.in.Close()
	if err != nil {
		return C.CString(err.Error())
	}
	err = info.out.Close()
	if err != nil {
		return C.CString(err.Error())
	}
	err = <-info.err
	if err != nil {
		return C.CString(err.Error())
	}
	return (*C.char)(C.NULL)
}

// Close discards a connection to Replicant.
//export Close
func Close(conn ConnectionID) {
	delete(connections, conn)
}

// cbufToByteSlice returns a []byte pointing at C memory.
// The returned slice must not be used after the C memory is freed.
func cbufToByteSlice(data unsafe.Pointer, len C.int) []byte {
	h := reflect.SliceHeader{uintptr(data), int(len), int(len)}
	return *(*[]byte)(unsafe.Pointer(&h))
}

func main() {
	// Empty main() needed for cgo.
}
