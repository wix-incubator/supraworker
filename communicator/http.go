package communicator

import (
	"bytes"
	"fmt"
	"strings"

	"context"
	"encoding/json"
	backoff "github.com/cenkalti/backoff/v4"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	config "github.com/wix/supraworker/config"
	utils "github.com/wix/supraworker/utils"
)

func init() {
	Constructors[ConstructorsTypeRest] = TypeSpec{
		instance:    NewRestCommunicator,
		constructor: NewConfiguredRestCommunicator,
		Summary: `
RestCommunicator is the default implementation of Communicator and is
used by Default.
Underline it is used for HTTP connections only.`,
		Description: `
It supports the following params:
- ` + "`method`" + ` http method more [in this document](https://golang.org/pkg/net/http/#pkg-constants)
- ` + "`headers`" + ` To make a request with custom headers
- ` + "`url`" + ` to send in a request for the given URL.`,
	}
}

// RestCommunicator represents HTTP communicator.
type RestCommunicator struct {
	Communicator
	section    string
	param      string
	method     string
	url        string
	headers    map[string]string
	configured bool
	mu         sync.RWMutex
}

// NewRestCommunicator prepare struct communicator for HTTP requests
func NewRestCommunicator() Communicator {
	return &RestCommunicator{}
}

// NewRestCommunicator prepare struct communicator for HTTP requests
func NewConfiguredRestCommunicator(section string) (Communicator, error) {
	comm := NewRestCommunicator()
	cfgParams := utils.ConvertMapStringToInterface(
		config.GetStringMapStringTemplated(section, config.CFG_PREFIX_COMMUNICATOR))
	if _, ok := cfgParams["section"]; !ok {
		cfgParams["section"] = fmt.Sprintf("%s.%s", section, config.CFG_PREFIX_COMMUNICATOR)
	}
	if _, ok := cfgParams["param"]; !ok {
		cfgParams["param"] = config.CFG_COMMUNICATOR_PARAMS_KEY
	}

	if err := comm.Configure(cfgParams); err != nil {
		return nil, err
	}

	return comm, nil
}

// Configured checks if RestCommunicator is configured.
func (s *RestCommunicator) Configured() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.configured
}

// Configure reads configuration properties from global configuration and
// from argument.
// NOTE: There are default headers
func (s *RestCommunicator) Configure(params map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if val, ok := params["section"]; ok {
		s.section = fmt.Sprintf("%v", val)
	}
	if val, ok := params["param"]; ok {
		s.param = fmt.Sprintf("%v", val)
	}
	if val, ok := params["method"]; ok {
		s.method = strings.ToUpper(fmt.Sprintf("%v", val))
	}
	if val, ok := params["url"]; ok {
		s.url = fmt.Sprintf("%v", val)
	}

	s.headers = map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
	if _, ok := params["headers"]; ok {
		if v, ok1 := params["headers"].(map[string]string); ok1 {
			for k1, v1 := range v {
				s.headers[k1] = v1
			}
		}

	}
	s.configured = true
	return nil
}

// Fetch metadata from external API with exponential backoff.
// Example for configuration:
//         fetch:
//             communicators:
//                 one:
//                     method: "get"
//                     backoff:
//                         maxelapsedtime: 600s
//                         maxinterval: 60s
//                         initialinterval: 10s

func (s *RestCommunicator) Fetch(ctx context.Context, params map[string]interface{}) (result []map[string]interface{}, err error) {
	try := 0
	operation := func() error {
		res, err := s.fetch(ctx, params)
		if err == nil {
			result = res
			//if try > 1 {
			//	utils.LoggerFromContext(utils.FromRetryID(ctx, try), log).Tracef("Successfully updated %s [%s]", s.url, s.method)
			//}
		} else {
			utils.LoggerFromContext(utils.FromRetryID(ctx, try), log).Tracef("Retrying %s [%s]", s.url, s.method)
		}
		try += 1
		return err
	}
	expBackoff := backoff.NewExponentialBackOff()
	backoffSection := fmt.Sprintf("%v.%v", s.section, config.CFG_PREFIX_BACKOFF)

	if val := config.GetTimeDurationDefault(backoffSection,
		config.CFG_PREFIX_BACKOFF_INITIALINTERVAL, time.Second); val.Milliseconds() > 0 {
		expBackoff.InitialInterval = val
	}
	if val := config.GetTimeDurationDefault(backoffSection,
		config.CFG_PREFIX_BACKOFF_MAXINTERVAL, time.Second); val.Milliseconds() > 0 {
		expBackoff.MaxInterval = val
	}
	if val := config.GetTimeDurationDefault(backoffSection,
		config.CFG_PREFIX_BACKOFF_MAXELAPSEDTIME, time.Second); val.Milliseconds() > 0 {
		expBackoff.MaxElapsedTime = val
	}

	//var notifyFunc backoff.Notify = func(e error, duration time.Duration) {
	//	utils.LoggerFromContext(ctx, log).Tracef("Retry on... %s", e)
	//}
	//errRetry := backoff.RetryNotify(operation, expBackoff, notifyFunc)
	errRetry := backoff.Retry(operation, expBackoff)

	return result, errRetry

}
func (s *RestCommunicator) fetch(ctx context.Context, params map[string]interface{}) (result []map[string]interface{}, err error) {
	var jsonStr []byte
	var req *http.Request
	var rawResponse map[string]interface{}

	allowedResponseCodes := config.GetIntSlice(s.section,
		config.CFG_PREFIX_ALLOWED_RESPONSE_CODES, []int{200, 201, 202})
	requestTimeout := DefaultRequestTimeout
	ctxReq := context.Background()
	if ctx != nil {
		if v := ctx.Value(CtxAllowedResponseCodes); v != nil {
			if val, ok := v.([]int); ok {
				allowedResponseCodes = val
			}
		}
		if v := ctx.Value(CtxRequestTimeout); v != nil {
			if duration, errParseDuration := time.ParseDuration(fmt.Sprintf("%v", v)); errParseDuration == nil {
				requestTimeout = duration
			}
		}
	}
	if requestTimeout < 1 {
		requestTimeout = DefaultRequestTimeout
	}
	ctxReqCancel, cancel := context.WithTimeout(ctxReq, requestTimeout)
	defer cancel()

	from := map[string]string{
		"ClientId":      config.C.ClientId,
		"ClusterId":     config.C.ClusterId,
		"ClusterPool":   config.C.ClusterPool,
		"ConfigVersion": config.C.ConfigVersion,
		"URL":           config.C.URL,
		"NumActiveJobs": fmt.Sprintf("%v", config.C.NumActiveJobs),
		"NumFreeSlots":  fmt.Sprintf("%v", config.C.NumFreeSlots),
		"NumWorkers":    fmt.Sprintf("%v", config.C.NumWorkers),
	}
	for k, v := range params {
		if v1, ok := v.(string); ok {
			from[k] = v1
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.url) < 1 {
		return nil, nil
	}
	allParams := config.GetStringMapStringTemplatedFromMap(s.section, s.param, from)
	//utils.LoggerFromContext(ctx, log).Infof("[%s] %v\ns.section %v , %v, \nfrom: %v", s.url, allParams, s.section, s.param,from)

	for k, v := range params {
		if v1, ok := v.(string); ok {
			allParams[k] = v1
		}
	}

	if len(allParams) > 0 {
		jsonStr, err = json.Marshal(&allParams)
		if err != nil {
			utils.LoggerFromContext(ctx, log).Tracef("\nFailed to marshal request %s  to %s \nwith %s\n", s.method, s.url, jsonStr)
			return nil, fmt.Errorf("%w due %s", ErrFailedMarshalRequest, err)
		}

		req, err = http.NewRequestWithContext(ctxReqCancel, s.method, s.url, bytes.NewBuffer(jsonStr))
	} else {
		req, err = http.NewRequestWithContext(ctxReqCancel, s.method, s.url, nil)
	}
	if err != nil {
		return result, fmt.Errorf("%w due %s", ErrFailedSendRequest, err)
	}
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("%w got %s", ErrFailedSendRequest, err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w got %s", ErrFailedReadResponseBody, err)
	}

	if !utils.ContainsInts(allowedResponseCodes, resp.StatusCode) {
		return nil, fmt.Errorf("%w %v in %v got body %s", ErrNotAllowedResponseCode, resp.StatusCode, allowedResponseCodes, body)
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		err = json.Unmarshal(body, &rawResponse)
		if err != nil {
			return nil, fmt.Errorf("%w from body %s got %s", ErrFailedUnmarshalResponse, body, err)
		}
		result = append(result, rawResponse)
	}

	return result, nil

}
