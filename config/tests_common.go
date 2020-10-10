package config

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"go.uber.org/goleak"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func LoadCfgForTests(t *testing.T, CfgFile string) (Config, Config) {
	tmp := C

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
