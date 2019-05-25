package chk

import "fmt"

func Fail(msg string, params ...interface{}) {
	panic(fmt.Sprintf(msg, params...))
}

func True(cond bool, msg string, params ...interface{}) {
	if !cond {
		Fail(msg, params...)
	}
}

func False(cond bool, msg string, params ...interface{}) {
	True(!cond, msg, params...)
}

func Equal(expected interface{}, actual interface{}) {
	if expected != actual {
		Fail("Expected %#v, got: %#v - %s", expected, actual)
	}
}

func NotNil(v interface{}) {
	if v == nil {
		Fail("Expected non-nil value, but was: %#v", v)
	}
}

func NoError(err error) {
	if err != nil {
		Fail("Unexpected error: %#v", err)
	}
}
