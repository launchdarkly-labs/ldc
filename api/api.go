package api

import (
	"context"
	"net/http"

	ldapi "github.com/launchdarkly/api-client-go"
)

var Auth context.Context
var Client *ldapi.APIClient

var CurrentToken string
var CurrentServer string
var CurrentProject string
var CurrentEnvironment string

type LoggingTransport struct{}

func (lt *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)

	// TODO this is bad, don't do this
	//resp.Body = ioutil.NopCloser(io.TeeReader(resp.Body, os.Stdout))
	return resp, err
}

func init() {
	Client = ldapi.NewAPIClient(&ldapi.Configuration{
		HTTPClient: &http.Client{
			Transport: &LoggingTransport{},
		},
		UserAgent: "ldc/0.0.1/go",
	})
}

// TODO
func SetServer(newServer string) {
	CurrentServer = newServer
	Client = ldapi.NewAPIClient(&ldapi.Configuration{
		BasePath: newServer,
		HTTPClient: &http.Client{
			Transport: &LoggingTransport{},
		},
		UserAgent: "ldc/0.0.1/go",
	})

}

func SetToken(newToken string) {
	CurrentToken = newToken
	Auth = context.WithValue(context.Background(), ldapi.ContextAPIKey, ldapi.APIKey{
		Key: newToken,
	})
}
