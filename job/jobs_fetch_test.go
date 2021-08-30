package job

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"github.com/wix/supraworker/model"
	"github.com/wix/supraworker/model/cmdtest"
	"go.uber.org/goleak"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// An ApiJobResponse represents a Job response.
// Example response
// {
//   "job_id": "dbd618f0-a878-e477-7234-2ef24cb85ef6",
//   "jobStatus": "RUNNING",
//   "has_error": false,
//   "error_msg": "",
//   "run_uid": "0f37a129-eb52-96a7-198b-44515220547e",
//   "job_name": "Untitled",
//   "cmd": "su  - hadoop -c 'hdfs ls ''",
//   "parameters": [],
//   "createDate": "1583414512",
//   "lastUpdated": "1583415483",
//   "stopDate": "1586092912",
//   "extra_run_id": "scheduled__2020-03-05T09:21:40.961391+00:00"
// }
type ApiJobResponse struct {
	JobId       string   `json:"job_id"`
	JobStatus   string   `json:"jobStatus"`
	JobName     string   `json:"job_name"`
	RunUID      string   `json:"run_uid"`
	ExtraRunUID string   `json:"extra_run_id"`
	CMD         string   `json:"cmd"`
	Parameters  []string `json:"parameters"`
	CreateDate  string   `json:"createDate"`
	LastUpdated string   `json:"lastUpdated"`
	StopDate    string   `json:"stopDate"`
	EnvVar      []string `json:"env"`
}

//func init() {
//	cmdtest.StartTrace()
//}
func TestHelperProcess(t *testing.T) {
	cmdtest.TestHelperProcess(t)
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestGenerateJobs(t *testing.T) {
	defer goleak.VerifyNone(t)

	// startTrace()
	want := "{\"job_uid\":\"job-testing.(*common).Name-fm\",\"run_uid\":\"1\",\"extra_run_id\":\"1\",\"msg\":\"'S'\\n\"}"
	var got string
	CMD := "echo && exit 0"
	responses := []ApiJobResponse{
		{
			JobId:       "job_id",
			JobStatus:   "PENDING",
			JobName:     "job_name",
			RunUID:      "run_uid",
			ExtraRunUID: "extra_run_id",
			CMD:         CMD,
			Parameters:  []string{},
			CreateDate:  "createDate",
			LastUpdated: "lastUpdated",
			StopDate:    "stopDate",
			EnvVar:      []string{"EnvVar=1"},
		},
		{
			JobId:       "job_id",
			JobStatus:   "PENDING",
			JobName:     "job_name",
			RunUID:      "run_uid",
			ExtraRunUID: "extra_run_id",
			CMD:         CMD,
			Parameters:  []string{},
			CreateDate:  "createDate",
			LastUpdated: "lastUpdated",
			StopDate:    "stopDate",
			EnvVar:      []string{"EnvVar=1"},
		},
	}

	notifyStdoutSent := make(chan bool)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var c ApiJobResponse
		if len(responses) > 1 {
			c, responses = responses[0], responses[1:]
		} else if len(responses) == 1 {
			c = responses[0]
		}
		c1 := make([]ApiJobResponse, 0)
		c1 = append(c1, c)
		js, err := json.Marshal(&c1)
		if err != nil {
			log.Tracef("Failed to marshal for '%v' due %v", c, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, errWrite := w.Write(js); errWrite != nil {
			t.Errorf("Can't w.Write %v due %v\n", js, err)
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Errorf("ReadAll %s", err)
		}
		got = fmt.Sprintf("%s", b)
		notifyStdoutSent <- true
	}))
	defer func() {
		srv.Close()
		model.FetchNewJobAPIURL = ""
		// restoreLevel()
	}()
	viper.SetConfigType("yaml")
	var yamlExample = []byte(`
    logs:
      update:
        method: GET
    jobs:
      get:
        url: "` + srv.URL + `"
        method: POST
    `)

	_ = viper.ReadConfig(bytes.NewBuffer(yamlExample))

	model.FetchNewJobAPIURL = srv.URL
	log.Trace(fmt.Sprintf("model.FetchNewJobAPIURL  %s", model.FetchNewJobAPIURL))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // cancel when we are getting the kill signal or exit
	jobs := make(chan *model.Job, 1)

	go func() {
		if err := StartGenerateJobs(ctx, jobs, time.Duration(150)*time.Millisecond, 30*time.Second); err != nil {
			log.Infof("StartGenerateJobs failed %v", err)
		}
	}()

	select {
	case <-notifyStdoutSent:
		log.Trace("notifyStdoutSent")
	case <-time.After(1 * time.Second):
		t.Errorf("timed out")
	}
	for job := range jobs {
		time.Sleep(50 * time.Millisecond)
		if job.Status != model.JOB_STATUS_PENDING {
			t.Errorf("Expected %s, got %s", model.JOB_STATUS_PENDING, job.Status)
		}

		if job.CMD != CMD {
			t.Errorf("want %s, got %s", want, got)
		}
		job.Status = model.JOB_STATUS_CANCELED
		// stop loop
		if len(responses) == 1 {
			cancel()
		}
		foundEnv := false
		time.Sleep(100 * time.Millisecond)
		for _, v := range job.CmdENV {
			if "EnvVar=1" == v {
				foundEnv = true
			}
		}
		if !foundEnv {
			t.Errorf("Expected found %v, got %v", true, foundEnv)
		}

	}
}
