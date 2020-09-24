package config

import (
	"go.uber.org/goleak"
	"testing"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

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
