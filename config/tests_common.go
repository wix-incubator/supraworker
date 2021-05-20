package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"go.uber.org/goleak"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func StringToCfgForTests(t *testing.T, yaml []byte) (Config, Config) {
	tmp := C
	C = Config{}
	viper.Reset()
	viper.SetConfigType("yaml")

	if err := viper.ReadConfig(bytes.NewBuffer(yaml)); err != nil {
		t.Errorf("Can't read config: %s\ngot %s", yaml, err)
	}
	err := viper.Unmarshal(&C)
	if err != nil {
		t.Errorf("unable to decode into struct, %v", err)

	}
	if C.ClientId == string("") {
		t.Errorf("Expected C.ClientId not empty got %v\n", C)
	}
	if C.ConfigVersion == string("") {
		t.Errorf("Expected C.ConfigVersion not empty got %v\n", C)
	}
	return C, tmp
}
func LoadCfgForTests(t *testing.T, CfgFile string) (Config, Config) {
	tmp := C
	viper.Reset()
	C = Config{}
	viper.SetConfigFile(CfgFile)
	// t.Logf("Loaded: %v", CfgFile)
	if err := viper.ReadInConfig(); err != nil {
		t.Errorf("Can't read config: %v", err)
	}
	err := viper.Unmarshal(&C)
	if err != nil {
		t.Errorf("unable to decode into struct, %v", err)

	}

	if C.ClientId == string("") {
		t.Errorf("Expected C.ClientId not empty got %v\n", C)
	}
	if C.ConfigVersion == string("") {
		t.Errorf("Expected C.ConfigVersion not empty got %v\n", C)
	}
	return C, tmp
}

func NewTestServer(t *testing.T, in func() interface{}, out func(string)) *httptest.Server {
	// Response server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := in()
		js, err := json.Marshal(&c)
		if err != nil {
			log.Warningf("Failed to marshal for '%v' due %v", c, err)
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
		out(fmt.Sprintf("%s", b))
	}))
	return srv
}

func NewFlakyTestServer(t *testing.T, out <-chan string, isFlakyNow <-chan bool, maxTimeout time.Duration) *httptest.Server {
	//s1 := rand.NewSource(time.Now().UnixNano())
	//r1 := rand.New(s1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		select {
		case <-isFlakyNow:
			http.Error(w, "Flaky response ", http.StatusInternalServerError)
			return
		default:
		}

		//if r1.Intn(2) != 0 {
		//	t.Log("Flaky response")
		//	http.Error(w, "Flaky response ", http.StatusInternalServerError)
		//	return
		//}

		var response []byte
		var gotBody map[string]interface{}
		var expectInBody map[string]interface{}
		select {
		case data := <-out:
			response = []byte(data)
			//t.Logf("Reading out %v", data)

		case <-time.After(maxTimeout):
			t.Fatalf("timed out")
		}

		byt, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll %s", err)
		}
		if err := json.Unmarshal(byt, &gotBody); err != nil {
			t.Fatalf("Cannot unmarsharl to gotBody %v got %v", byt, err)
		}
		if err := json.Unmarshal(response, &expectInBody); err != nil {
			t.Fatalf("Cannot unmarsharl to expectInBody %v got %v", byt, err)
		}
		for k, v := range expectInBody {
			valInBody, ok := gotBody[k]
			if !ok {
				t.Fatalf("Key '%s' not exists in body", k)
			} else if fmt.Sprintf("%v", valInBody) != fmt.Sprintf("%v", v) {
				t.Fatalf("Body '%v' != expected '%v'", valInBody, v)
			}

		}

		js, err := json.Marshal(&response)
		if err != nil {
			log.Errorf("Failed to marshal for '%v' due %v", response, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		if _, errWrite := w.Write(response); errWrite != nil {
			t.Fatalf("Can't w.Write %v due %v\n", js, err)
		}
		w.WriteHeader(http.StatusOK)

	}))
	return srv
}
