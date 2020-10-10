package communicator

import (
	epsagonhttp "github.com/epsagon/epsagon-go/wrappers/net/http"
	"net/http"
    "time"
)

func SetHttpClientWrapper(f func() *http.Client) {

	globalHttpClient = f()
}

func epsagonClient() *http.Client {
	return &http.Client{
		Transport: epsagonhttp.NewTracingTransport(),
		Timeout:   time.Duration(15 * time.Second),
	}
}
func SetEpsagonHttpWrapper() {
	SetHttpClientWrapper(epsagonClient)
}
