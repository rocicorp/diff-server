package db

import (
	"fmt"
	"log"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/spec"
)

type Progress struct {
	ChunksToTransmit   uint64
	ChunksTrasmitted   uint64
	ApproxBytesWritten uint64
}

// HackyShallowSync bypasses history and syncs only the latest value from remote.
// Because there is no history, this breaks bidirectional sync.
func (db *DB) HackyShallowSync(remote spec.Spec, progress func(p Progress)) error {
	pchan := make(chan datas.PullProgress)
	go func() {
		for p := range pchan {
			log.Println("pull progress", p)
			if progress != nil {
				progress(Progress{p.KnownCount, p.DoneCount, p.ApproxWrittenBytes})
			}
		}
	}()

	remoteDB, err := Load(remote, fmt.Sprintf("%s/remote", db.origin))
	if err != nil {
		return err
	}

	r := remoteDB.noms.WriteValue(remoteDB.head.Original.Get("value"))
	// Despite the fact that it takes databases as arguments, Pull() actually deals in
	// ChunkStores so it won't find values that are in-memory in the database but not yet
	// flushed. 🤷‍♂️.
	remoteDB.noms.Flush()
	datas.Pull(remoteDB.noms, db.noms, r, pchan)

	v := db.noms.ReadValue(r.TargetHash())
	var c Commit
	err = marshal.Unmarshal(v, &c.Value)
	if err != nil {
		return err
	}

	db.noms.SetHead(db.noms.GetDataset(LOCAL_DATASET), db.noms.WriteValue(marshal.MustMarshal(db.noms, c)))
	return nil
}
