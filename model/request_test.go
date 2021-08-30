package model

import (
	"fmt"
	"github.com/wix/supraworker/config"
	"testing"
)

func TestGetParamsFromSection(t *testing.T) {
	clientId := "ClientId"
	jobId := "job_id"
	keys := map[string]string{"job": jobId, "client": clientId}

	var yamlString = []byte(`
    ClientId: "` + clientId + `"
    version: "1.0"
    jobs:
      run: &run
        communicator: &comm
          params:
            client: "{{ .ClientId }}"
            job: "` + jobId + `"
      failed: &failed
        communicator:
          <<: *comm
    `)

	C, tmpC := config.StringToCfgForTests(t, yamlString)
	config.C = C
	defer func() {
		config.C = tmpC
	}()
	cases := []struct {
		section string
	}{
		{
			section: "run",
		},
		{
			section: "failed",
		},
	}
	for _, tc := range cases {
		res := GetParamsFromSection(fmt.Sprintf("jobs.%s.communicator", tc.section), "params")
		if len(res) < len(keys) {
			t.Fatalf("section %s length '%d' < '%d'", tc.section, len(res), len(keys))
		}
		for k, v := range keys {
			if val, ok := res[k]; !ok {
				t.Fatalf("res[\"%s\"] is empty", k)
			} else if val != v {
				t.Fatalf("res[\"%s\"] '%s' != '%s'", k, val, v)
			}
		}

	}
}
