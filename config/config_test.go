package config

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tmp := C
	defer func() {
		C = tmp
	}()
	C = Config{}
	CfgFile = "fixtures/test_load.yaml"
	initConfig()
	t.Logf("Loaded: %v", CfgFile)
	if C.ClientId == string("") {
		t.Errorf("Expected C not empty got %v\n", C)
	}
}
