package db

import (
	"fmt"
	"testing"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
)

func TestShallowSync(t *testing.T) {
	assert := assert.New(t)

	remote, rdir := LoadTempDB(assert)
	local, ldir := LoadTempDB(assert)

	fmt.Println("remote", rdir)
	fmt.Println("local", ldir)

	for i := 0; i < 10; i++ {
		err := remote.Put(fmt.Sprintf("k%d", i), types.String(fmt.Sprintf("v%d", i)))
		assert.NoError(err)
	}

	count := 0
	progress := func(p Progress) {
		count++
	}
	rspec, err := spec.ForDatabase(rdir)
	assert.NoError(err)
	err = local.HackyShallowSync(rspec, progress)
	assert.NoError(err)
	assert.True(count > 0)
}
