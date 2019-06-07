package history

import (
	"fmt"
	"testing"
	"time"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	assert := assert.New(t)
	ts := (&chunks.TestStorage{}).NewView()
	noms := types.NewValueStore(ts)

	type commit struct {
		Parents []types.Ref
		Value   string
	}

	makeCommit := func(parents ...types.Value) types.Value {
		pr := make([]types.Ref, len(parents))
		for i, p := range parents {
			pr[i] = types.NewRef(p)
		}
		r := marshal.MustMarshal(noms, commit{
			Parents: pr,
			Value:   fmt.Sprintf("%s", time.Now()),
		})
		noms.WriteValue(r)
		return r
	}

	c0 := makeCommit()

	cache := NewCache(noms)
	assert.False(cache.Has(c0.Hash()))

	err := cache.Populate(c0.Hash())
	assert.NoError(err)
	assert.True(cache.Has(c0.Hash()))

	err = cache.Populate(c0.Hash())
	assert.NoError(err)
	assert.True(cache.Has(c0.Hash()))

	c1 := makeCommit(c0)
	c2 := makeCommit(c1)

	err = cache.Populate(c2.Hash())
	assert.NoError(err)
	assert.True(cache.Has(c1.Hash()))
	assert.True(cache.Has(c2.Hash()))

	c3 := makeCommit(c2)
	c3b := makeCommit(c2)
	c4 := makeCommit(c3)
	c5 := makeCommit(c3b, c4)
	err = cache.Populate(c5.Hash())
	assert.NoError(err)
	assert.True(cache.Has(c3.Hash()))
	assert.True(cache.Has(c3b.Hash()))
	assert.True(cache.Has(c4.Hash()))
	assert.True(cache.Has(c5.Hash()))
}
