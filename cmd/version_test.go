package cmd

import (
	"testing"
)

func TestFormattedVersion(t *testing.T) {

	version := FormattedVersion()
	if version == "" {
		t.Errorf("Expected version not be empty. Got %s", version)
	}

}
