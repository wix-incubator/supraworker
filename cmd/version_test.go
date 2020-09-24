package cmd

import (
	"go.uber.org/goleak"
	"testing"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestFormattedVersion(t *testing.T) {

	version := FormattedVersion()
	if version == "" {
		t.Errorf("Expected version not be empty. Got %s", version)
	}

}
