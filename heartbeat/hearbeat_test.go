package heartbeat

import (
	"bytes"
	"context"
	"fmt"
	"github.com/spf13/viper"
	config "github.com/wix/supraworker/config"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var backup config.Config

func init() {
	backup = config.C
}

func testTeardown() {
	config.C = backup
}

func TestHeartBeatApi(t *testing.T) {

	want := "asdasd"
	var got string
	notifyStdoutSent := make(chan bool)
	ctx, cancel := context.WithCancel(context.Background())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "{}")
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Errorf("ReadAll %s", err)
		}
		got = string(fmt.Sprintf("%s", b))
		notifyStdoutSent <- true
		cancel()
	}))
	defer func() {
		srv.Close()
		cancel()
		testTeardown()
	}()
	viper.SetConfigType("yaml")
	var yamlExample = []byte(`
    heartbeat:
        enable: true
        communicators:
            default:
                url: "` + srv.URL + `"
                method: post
                params:
                  "id": "` + want + `"
    `)

	if err := viper.ReadConfig(bytes.NewBuffer(yamlExample)); err != nil {
		t.Errorf("Can't read config: %v\n", err)
	}
	go func() {
		if err := StartHeartBeat(ctx, "heartbeat", 100*time.Millisecond); err != nil {
			log.Warnf("StartHeartBeat returned error %v", err)
		}
	}()

	select {
	case <-notifyStdoutSent:
		log.Debug("notifyStdoutSent")
		cancel()
	case <-time.After(5 * time.Second):
		t.Errorf("timed out")
		t.Errorf(got)
	}

	if got != "{\"id\":\""+want+"\"}" {
		t.Errorf("want %s, got %v", want, got)
	}
}
