package cmdtest

import "testing"

func TestNothing(t *testing.T) {
	if false {
		t.Fail()
	}
}
