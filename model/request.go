package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/weldpua2008/supraworker/communicator"
	"net"

	// "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"html/template"
	"io/ioutil"
	"net/http"
	// "strings"
	"time"
	// "runtime"

	config "github.com/weldpua2008/supraworker/config"
)

// GetAPIParamsFromSection of the configuration
func GetAPIParamsFromSection(stage string) map[string]string {
	return GetParamsFromSection(stage, "params")
}

// GetParamsFromSection from stage & with sub-param
// Example stage = `jobs.logstream` and param `params` in the following config:
// with `GetParamsFromSection("jobs.logstream", "params")`
// var yamlExample = []byte(`
// jobs:
//   logstream: &update
//     url: "localhost"
//     method: post
//     params:
//       "job_uid": "job_uid"
//       "run_uid": "1"
func GetParamsFromSection(stage string, param string) map[string]string {
	c := make(map[string]string)
	params := viper.GetStringMapString(fmt.Sprintf("%s.%s", stage, param))
	for k, v := range params {
		var tplBytes bytes.Buffer
		tpl, err := template.New("params").Parse(v)
		if err != nil {
			log.Tracef("stage %s params executing template: %s", stage, err)
			continue
		}
		err = tpl.Execute(&tplBytes, config.C)
		if err != nil {
			log.Tracef("params executing template: %s", err)
			continue
		}
		c[k] = tplBytes.String()
	}
	return c
}

// GetSliceParamsFromSection from stage & with sub-param
// Example stage = `jobs.logstream` and param `params` in the following config:
// with `GetParamsFromSection("jobs.logstream", "resend-params")`
// var yamlExample = []byte(`
// jobs:
//   logstream: &update
//     url: "localhost"
//     method: post
//     resend-params:
//      - "job_uid"
//      - "run_uid"
//      - "extra_run_id"
func GetSliceParamsFromSection(stage string, param string) []string {
	c := make([]string, 0)
	params := viper.GetStringSlice(fmt.Sprintf("%s.%s", stage, param))
	for _, v := range params {
		var tplBytes bytes.Buffer
		tpl, err := template.New("params").Parse(v)
		if err != nil {
			log.Tracef("stage %s params executing template: %s", stage, err)
			continue
		}
		err = tpl.Execute(&tplBytes, config.C)
		if err != nil {
			log.Tracef("params executing template: %s", err)
			continue
		}
		c = append(c, tplBytes.String())
	}
	return c
}

// DoApi REST calls via communicators
func DoApi(ctx context.Context, params map[string]interface{}, stage string) error {
	//defer func() {
	//	//log.Tracef("http.Client stop")
	//	log.Tracef("[DoApi] ctx %v params %v stage %v", ctx, params, stage)
	//}()
	section := fmt.Sprintf("%v.%v",
		config.CFG_PREFIX_JOBS, stage)
	if ctx == nil {
		ctx = context.Background()
	}
	defaultRequestTimeout := communicator.DefaultRequestTimeout
	if value := ctx.Value(CtxKeyRequestTimeout); value != nil {
		if duration, errParseDuration := time.ParseDuration(fmt.Sprintf("%v", value)); errParseDuration == nil {
			defaultRequestTimeout = duration
		} else {
			log.Tracef("[DoApi] Cannot parse duration %v stage %v", errParseDuration, stage)
		}
	}
	if defaultRequestTimeout < 1 {
		defaultRequestTimeout = communicator.DefaultRequestTimeout
	}
	allCommunicators, errGetCommunicators := communicator.GetCommunicatorsFromSection(section)
	if errGetCommunicators != nil {
		if comm, errGetCommunicators := communicator.GetSectionCommunicator(section); errGetCommunicators == nil {
			allCommunicators = []communicator.Communicator{comm}
		} else {
			log.Warnf("[DoApi] Cannot use communicators from %s got %s", section, errGetCommunicators)
		}
	}
	fetchCtx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel() // cancel when we are getting the kill signal or exit
	for _, comm := range allCommunicators {
		res, err := comm.Fetch(fetchCtx, params)
		//log.Infof("[comm.Fetch] params %v => %v", params, res)
		if err != nil {
			return fmt.Errorf("%w got %v error %v sent %v", ErrFailedSendRequest, res, err, params)
		}
	}
	return nil
}

// DoApiCall for the jobs stages
// TODO: add custom headers
func DoApiCall(ctx context.Context, params map[string]string, stage string) (error, []map[string]interface{}) {
	var rawResponseArray []map[string]interface{}
	url := viper.GetString(fmt.Sprintf("%s.url", stage))
	if len(url) < 1 {
		return fmt.Errorf("empty url on stage %s", stage), rawResponseArray
	}
	method := chooseHttpMethod(viper.GetString(fmt.Sprintf("%s.method", stage)), "POST")
	var rawResponse map[string]interface{}
	var req *http.Request
	var err error
	var jsonStr []byte
	defaultRequestTimeout := communicator.DefaultRequestTimeout
	ctxReq := context.Background()
	if ctx != nil {
		ctxReq = ctx
		if value := ctx.Value(CtxKeyRequestTimeout); value != nil {
			if duration, errParseDuration := time.ParseDuration(fmt.Sprintf("%v", value)); errParseDuration == nil {
				defaultRequestTimeout = duration
			} /* else {
				return fmt.Errorf("Cannot parse duration %v", errParseDuration), nil
			} */
		}
	}
	if defaultRequestTimeout < 1 {
		defaultRequestTimeout = communicator.DefaultRequestTimeout
	}
	ctxReqCancel, cancel := context.WithTimeout(ctxReq, defaultRequestTimeout)
	defer cancel()

	if len(params) > 0 {
		jsonStr, err = json.Marshal(&params)
		if err != nil {
			log.Trace(fmt.Sprintf("\nFailed to marshal request %s  to %s \nwith %s\n", method, url, jsonStr))
			return fmt.Errorf("Failed to marshal request due %s", err), nil
		}

		req, err = http.NewRequestWithContext(ctxReqCancel, method, url, bytes.NewBuffer(jsonStr))
	} else {
		req, err = http.NewRequestWithContext(ctxReqCancel, method, url, nil)
	}
	if err != nil {
		return fmt.Errorf("Failed to create request due %s", err), nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: defaultRequestTimeout}
	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if e, ok := err.(net.Error); ok && e.Timeout() {
		return fmt.Errorf("Do request timeout: %s", err), nil
	} else if err != nil {
		return fmt.Errorf("Failed to send request due %s", err), nil
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error read response body got %s", err), nil
	}
	if (resp.StatusCode > 202) || (resp.StatusCode < 200) {
		log.Tracef("\nMaking request %s  to %s \nwith %s\nStatusCode %d Response %s\n", method, url, jsonStr, resp.StatusCode, body)
	}
	err = json.Unmarshal(body, &rawResponseArray)
	if err != nil {
		err = json.Unmarshal(body, &rawResponse)
		if err != nil {
			return fmt.Errorf("error Unmarshal response: %s due %s", body, err), nil
		}
		rawResponseArray = append(rawResponseArray, rawResponse)
	}
	return nil, rawResponseArray

}

// NewRemoteApiRequest perform request to API.
func NewRemoteApiRequest(ctx context.Context, section string, method string, url string) (error, []map[string]interface{}) {
	var rawResponseArray []map[string]interface{}
	var rawResponse map[string]interface{}

	t := viper.GetStringMapString(section)
	reqSendJsonMap := make(map[string]string)
	for k, v := range t {
		var tplBytes bytes.Buffer
		tpl, err := template.New("params").Parse(v)
		if err != nil {
			log.Warn("executing template:", err)
			continue
		}
		err = tpl.Execute(&tplBytes, config.C)
		if err != nil {
			log.Warn("executing template:", err)
		}
		reqSendJsonMap[k] = tplBytes.String()
	}
	var req *http.Request
	var err error
	defaultRequestTimeout := communicator.DefaultRequestTimeout
	ctxReq := context.Background()
	if ctx != nil {
		ctxReq = ctx
		if value := ctx.Value(CtxKeyRequestTimeout); value != nil {
			if duration, errParseDuration := time.ParseDuration(fmt.Sprintf("%v", value)); errParseDuration == nil {
				defaultRequestTimeout = duration
			} /* else {
				return fmt.Errorf("Cannot parse duration %v", errParseDuration), nil
			} */
		}
	}
	if defaultRequestTimeout < 1 {
		defaultRequestTimeout = communicator.DefaultRequestTimeout
	}
	ctxReqCancel, cancel := context.WithTimeout(ctxReq, defaultRequestTimeout)
	defer cancel()

	if len(reqSendJsonMap) > 0 {
		reqJsonStr, errMarsh := json.Marshal(&reqSendJsonMap)

		if errMarsh != nil {
			return fmt.Errorf("Failed to marshal request due %s", errMarsh), nil
		}
		req, err = http.NewRequestWithContext(ctxReqCancel, method, url, bytes.NewBuffer(reqJsonStr))
	} else {
		req, err = http.NewRequestWithContext(ctxReqCancel, method, url, nil)
	}
	if err != nil {
		return fmt.Errorf("Failed to create new request due %s", err), nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: defaultRequestTimeout}
	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if e, ok := err.(net.Error); ok && e.Timeout() {
		return fmt.Errorf("Do request timeout: %s", err), nil
	} else if err != nil {
		return fmt.Errorf("Failed to send request due %s", err), nil
	}
	if body, err := ioutil.ReadAll(resp.Body); err == nil {
		if (resp.StatusCode > 202) || (resp.StatusCode < 200) {
			log.Tracef("StatusCode %d Response %s", resp.StatusCode, body)
		}
		if err = json.Unmarshal(body, &rawResponseArray); err != nil {
			if err = json.Unmarshal(body, &rawResponse); err != nil {
				return fmt.Errorf("error Unmarshal response: %s due %s", body, err), nil
			}
			rawResponseArray = append(rawResponseArray, rawResponse)
		}
	} else {
		return fmt.Errorf("error read response body got %s", err), nil
	}

	return nil, rawResponseArray

}
