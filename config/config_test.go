package config

import (
	"testing"
)

func TestLoadConfigClientId(t *testing.T) {
	tmp := C
	defer func() {
		C = tmp
	}()
	cases := []struct {
		CfgFile      string
		ClientId     string
		WantClientId string
	}{
		{
			CfgFile:      "fixtures/test_load.yaml",
			WantClientId: "clientId",
		},
		{
			CfgFile:      "fixtures/test_load.yaml",
			WantClientId: "ConsoleClientId",
			ClientId:     "ConsoleClientId",
		},
		{
			CfgFile:      "fixtures/test_default.yaml",
			WantClientId: "supraworker",
		},
		{
			CfgFile:      "fixtures/test_default.yaml",
			WantClientId: "ConsoleClientId",
			ClientId:     "ConsoleClientId",
		},
	}

	for _, tc := range cases {
		C = Config{}
		CfgFile = tc.CfgFile
		ClientId = tc.ClientId
		initConfig()

		if C.ClientId != tc.WantClientId {
			t.Logf("Loaded: %v", CfgFile)
			t.Errorf("Expected C.ClientId %s got %v\n", tc.WantClientId, C.ClientId)
		}
	}
}

func TestLoadConfigNumWorkers(t *testing.T) {
	tmp := C
	defer func() {
		C = tmp
	}()
	cases := []struct {
		CfgFile        string
		NumWorkers     int
		WantNumWorkers int
	}{
		{
			CfgFile:        "fixtures/test_load.yaml",
			WantNumWorkers: DefaultNumWorkers,
		},
		{
			CfgFile:        "fixtures/test_load.yaml",
			NumWorkers:     1, //Should override
			WantNumWorkers: 1,
		},
		{
			CfgFile:        "fixtures/test_num_workers.yaml",
			WantNumWorkers: 10,
		},
		{
			CfgFile:        "fixtures/test_num_workers.yaml",
			WantNumWorkers: 11,
			NumWorkers:     11, // Should override
		},
	}

	for _, tc := range cases {
		C = Config{}
		CfgFile = tc.CfgFile
		NumWorkers = tc.NumWorkers
		initConfig()

		if C.NumWorkers != tc.WantNumWorkers {
			t.Logf("Loaded: %v", CfgFile)
			t.Errorf("Expected C.NumWorkers %d got %d\n", tc.WantNumWorkers, C.NumWorkers)
		}
	}
}

func TestLoadConfigPrometheus(t *testing.T) {
	tmp := C
	defer func() {
		C = tmp
	}()
	cases := []struct {
		CfgFile                 string
		WantPrometheusNamespace string
		WantPrometheusService   string
	}{
		{
			CfgFile:                 "fixtures/test_prometheus.yaml",
			WantPrometheusNamespace: "test-prom",
			WantPrometheusService:   "test-service",
		},
	}

	for _, tc := range cases {
		C = Config{}
		CfgFile = tc.CfgFile
		initConfig()

		if C.PrometheusNamespace != tc.WantPrometheusNamespace {
			t.Logf("Loaded: %v", CfgFile)
			t.Errorf("Expected C.PrometheusNamespace %s got %s\n", tc.WantPrometheusNamespace, C.PrometheusNamespace)
		}
		if C.PrometheusService != tc.WantPrometheusService {
			t.Logf("Loaded: %v", CfgFile)
			t.Errorf("Expected C.PrometheusService %s got %s\n", tc.WantPrometheusService, C.PrometheusService)
		}
	}
}
