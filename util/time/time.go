package time

import (
	"time"

	"github.com/attic-labs/noms/go/util/datetime"
)

var (
	fakeTime *time.Time
)

func Now() time.Time {
	if fakeTime != nil {
		return *fakeTime
	}
	return time.Now()
}

func DateTime() datetime.DateTime {
	return datetime.DateTime{Now()}
}

func SetFake() (undo func()) {
	loc, err := time.LoadLocation("US/Hawaii")
	if err != nil {
		panic(err)
	}
	f := time.Date(2014, 1, 24, 0, 0, 0, 0, loc)
	fakeTime = &f
	return ClearFake
}

func ClearFake() {
	fakeTime = nil
}

func String(t time.Time) string {
	if fakeTime != nil {
		t = t.In(fakeTime.Location())
	}
	return t.String()
}
