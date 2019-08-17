package time

import (
	"testing"
	gt "time"

	"github.com/stretchr/testify/assert"
)

func TestBasics(t *testing.T) {
	assert := assert.New(t)
	SetFake()
	f := Now()
	assert.NotEqual(f, gt.Now())
	assert.NotEmpty(f)
	ClearFake()
	assert.NotEqual(f, Now())
	func() {
		defer SetFake()()
		assert.Equal(f, Now())
	}()
	assert.NotEqual(f, Now())
}
