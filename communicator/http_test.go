package communicator

import (
	"context"
	"errors"
	config "github.com/weldpua2008/supraworker/config"
	"strings"
	"testing"
)

type Response struct {
	Number int    `json:"number"`
	Str    string `json:"str"`
}

var (
	globalGot string
	responses []Response
)

func in() interface{} {
	var c Response
	if len(responses) > 1 {
		c, responses = responses[0], responses[1:]
	} else if len(responses) == 1 {
		c = responses[0]
	}
	c1 := make([]Response, 0)
	c1 = append(c1, c)
	return c1
}

func out(in string) {
	globalGot = in
}

func TestGetCommunicatorFetch(t *testing.T) {
	C, tmpC := config.LoadCfgForTests(t, "fixtures/fetch_http.yml")
	config.C = C

	defer func() {
		config.C = tmpC
	}()
	globalGot = ""
	responses = []Response{
		{
			Number: 1,
			Str:    "Str",
		},
		{
			Number: 2,
			Str:    "Str1",
		},
	}
	srv := config.NewTestServer(t, in, out)
	defer func() {
		globalGot = ""
		srv.Close()
	}()

	cases := []struct {
		section      string
		params       map[string]interface{}
		want         map[string]interface{}
		wantErr      error
		wantFetchErr error
		gotContains  []string
	}{
		{
			section: "get",
			params: map[string]interface{}{
				"method":   "GET",
				"url":      srv.URL,
				"clientId": "clientId",
			},
			want: map[string]interface{}{
				"Number": responses[0].Number,
				"Str":    responses[0].Str,
			},
			wantErr:      nil,
			wantFetchErr: nil,
			gotContains:  []string{`"a":"a"`, `"c":"c"`, `k":"clientId"`},
		},
		{
			section: "get",
			params: map[string]interface{}{
				"method": "GET",
				"url":    srv.URL,
			},
			want: map[string]interface{}{
				"Number":   responses[1].Number,
				"Str":      responses[1].Str,
				"clientId": "clientId",
			},
			wantErr:      nil,
			wantFetchErr: nil,
		},
	}
	for _, tc := range cases {
		result, got := GetSectionCommunicator(tc.section)
		if (tc.wantErr == nil) && (tc.wantErr != got) {
			t.Errorf("want %v, got %v", tc.wantErr, got)
		} else if (tc.want == nil) && (!result.Configured()) {
			t.Errorf("want %v, got %v, res %v", true, result.Configured(), result)
		} else {
			if !errors.Is(got, tc.wantErr) {
				t.Errorf("want %v, got %v, res %v", tc.wantErr, got, result)
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel() // cancel when we are getting the kill signal or exit
		sent_params := map[string]interface{}{
			"a": "a",
		}
		if err := result.Configure(tc.params); err != nil {
			t.Errorf("want %v, got %v, results %v", nil, err, result)
		}
		c, _ := result.(*RestCommunicator)

		if len(c.url) < 1 {
			t.Errorf("want url len, got %v", result)

		}

		ret, getFetchErr := result.Fetch(ctx, sent_params)
		if (tc.wantFetchErr == nil) && (tc.wantFetchErr != getFetchErr) {
			t.Errorf("want %v, got %v", tc.wantFetchErr, getFetchErr)
			// WARNING: Keys are always in Lower case.
		} else if (tc.wantFetchErr == nil) && (ret[0]["str"] != tc.want["Str"]) {
			t.Errorf("want %v, got %v", tc.want["Str"], ret[0]["str"])
		} else {
			if !errors.Is(getFetchErr, tc.wantFetchErr) {
				t.Errorf("want %v, got %v, res %v", tc.wantFetchErr, getFetchErr, result)
			}
		}
		if len(globalGot) < 1 {
			t.Errorf("want len > 0 , got %v, send params %v", globalGot, sent_params)
		}
		for _, expectGot := range tc.gotContains {
			if !strings.Contains(globalGot, expectGot) {
				t.Errorf("want %v in got %v", expectGot, globalGot)
			}

		}

	}
}

func TestGetCommunicatorsFromSectionFetch(t *testing.T) {
	C, tmpC := config.LoadCfgForTests(t, "fixtures/fetch_http.yml")
	config.C = C

	defer func() {
		config.C = tmpC
	}()
	responses = []Response{
		{
			Number: 1,
			Str:    "Str",
		},
		{
			Number: 2,
			Str:    "Str1",
		},
	}
	// Response server.
	srv := config.NewTestServer(t, in, out)
	defer func() {
		srv.Close()
	}()

	cases := []struct {
		section      string
		params       map[string]interface{}
		want         map[string]interface{}
		sent_params  map[string]interface{}
		wantErr      error
		wantFetchErr error
		gotContains  []string
	}{
		{
			section: "GetCommunicatorsFromSection.get",
			sent_params: map[string]interface{}{
				"a": "a", "Param": "Param",
			},
			params: map[string]interface{}{
				"url":   srv.URL,
				"Param": "Param",
			},
			want: map[string]interface{}{
				"Number": responses[0].Number,
				"Str":    responses[0].Str,
			},
			wantErr:      nil,
			wantFetchErr: nil,
			gotContains: []string{`"a":"a"`, `"c":"c"`, `key":"clientId"`,
				`param":"Param"`},
		},
		{
			section: "GetCommunicatorsFromSection.get",
			params: map[string]interface{}{
				"url": srv.URL,
			},
			sent_params: map[string]interface{}{
				"a": "a",
			},
			want: map[string]interface{}{
				"Number": responses[1].Number,
				"Str":    responses[1].Str,
			},
			wantErr:      nil,
			wantFetchErr: nil,
		},
	}
	for _, tc := range cases {
		results, got := GetCommunicatorsFromSection(tc.section)
		var result Communicator

		for _, val := range results {
			result = val
		}

		if (tc.wantErr == nil) && (tc.wantErr != got) {
			t.Errorf("want %v, got %v", tc.wantErr, got)
		} else if (tc.want == nil) && (!result.Configured()) {
			t.Errorf("want %v, got %v, res %v", true, result.Configured(), result)
		} else {
			if !errors.Is(got, tc.wantErr) {
				t.Errorf("want %v, got %v, res %v", tc.wantErr, got, result)
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel() // cancel when we are getting the kill signal or exit
		// sent_params := map[string]interface{}{
		// 	"a": "a",
		// }

		if err := result.Configure(tc.params); err != nil {
			t.Errorf("want %v, got %v, results %v", nil, err, result)
		}
		c, _ := result.(*RestCommunicator)

		if len(c.url) < 1 {
			t.Errorf("want url len, got %v", result)

		}

		ret, getFetchErr := result.Fetch(ctx, tc.sent_params)
		if (tc.wantFetchErr == nil) && (tc.wantFetchErr != getFetchErr) {
			t.Errorf("want %v, got %v", tc.wantFetchErr, getFetchErr)
			// WARNING: Keys are always in Lower case.
		} else if (tc.wantFetchErr == nil) && (ret[0]["str"] != tc.want["Str"]) {
			t.Errorf("want %v, got %v", tc.want["Str"], ret[0]["str"])
		} else {
			if !errors.Is(getFetchErr, tc.wantFetchErr) {
				t.Errorf("want %v, got %v, res %v", tc.wantFetchErr, getFetchErr, result)
			}
		}
		if len(globalGot) < 1 {
			t.Errorf("want len > 0 , got %v, send params %v", globalGot, tc.sent_params)
		}
		for _, expectGot := range tc.gotContains {
			if !strings.Contains(globalGot, expectGot) {
				t.Errorf("want %v in got %v", expectGot, globalGot)
			}

		}

	}
}
