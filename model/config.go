package model

import (
	"github.com/spf13/viper"
	"net/http"
	"strings"
)

// func init() {
//     viper.WatchConfig()
//     viper.OnConfigChange(func(e fsnotify.Event) {
//     	log.Trace("Config file changed:", e.Name)
//         ReinitializeConfig()
//     })
// }

func chooseHttpMethod(provided string, def string) string {
	// log.Tracef("provided %v",provided)
	provided = strings.ToUpper(provided)
	switch provided {
	case http.MethodGet:
	case http.MethodHead:
	case http.MethodPost:
	case http.MethodPut:
	case http.MethodPatch:
	case http.MethodDelete:
	case http.MethodConnect:
	case http.MethodOptions:
	case http.MethodTrace:

	default:
		provided = def
	}
	return provided

}

// ReinitializeConfig on load or config file changes
func ReinitializeConfig() error {
	if len(viper.GetString("logs.update.url")) > 0 {
		StreamingAPIURL = viper.GetString("logs.update.url")

	}
	StreamingAPIMethod = chooseHttpMethod(viper.GetString("logs.update.method"), http.MethodPost)
	if len(viper.GetString("jobs.get.url")) > 0 {
		FetchNewJobAPIURL = viper.GetString("jobs.get.url")
	}
	FetchNewJobAPIMethod = chooseHttpMethod(viper.GetString("jobs.get.method"), http.MethodPost)

	return nil

}
