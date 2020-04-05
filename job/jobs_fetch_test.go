package job

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"github.com/weldpua2008/supraworker/model"
	"github.com/weldpua2008/supraworker/model/cmdtest"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func init() {
	cmdtest.StartTrace()
}
func TestHelperProcess(t *testing.T) {
	cmdtest.TestHelperProcess(t)
}

func TestGenerateJobs(t *testing.T) {

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
		},
	}

	// notifyStdoutSent := make(chan bool)
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
		w.Write(js)
		// w.WriteHeader(200)

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Errorf("ReadAll %s", err)
		}
		got = string(fmt.Sprintf("%s", b))
		// notifyStdoutSent <- true
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

	viper.ReadConfig(bytes.NewBuffer(yamlExample))

	model.FetchNewJobAPIURL = srv.URL
	log.Trace(fmt.Sprintf("model.FetchNewJobAPIURL  %s", model.FetchNewJobAPIURL))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // cancel when we are getting the kill signal or exit
	jobs := make(chan *model.Job, 1)

	go StartGenerateJobs(jobs, ctx, time.Duration(5)*time.Second)

	for job := range jobs {
		if job.Status != model.JOB_STATUS_PENDING {
			t.Errorf("Expected %s, got %s", model.JOB_STATUS_PENDING, job.Status)
		}

		if job.CMD != CMD {
			t.Errorf("want %s, got %v", want, got)
		}
		job.Status = model.JOB_STATUS_CANCELED
		// stop loop
		if len(responses) == 1 {
			cancel()
		}

	}
}
