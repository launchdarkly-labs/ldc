package api

import (
	"context"
	"net/http"

	"github.com/launchdarkly/api-client-go"
)

var Auth context.Context
var Client *ldapi.APIClient

const DefaultServer = "https://app.launchdarkly.com/api/v2"

var CurrentToken string
var CurrentServer string
var CurrentProject = "default"
var CurrentEnvironment = "production"

type LoggingTransport struct{}

func (lt *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)

	// TODO this is bad, don't do this
	//resp.Body = ioutil.NopCloser(io.TeeReader(resp.Body, os.Stdout))
	return resp, err
}

var HttpClient *http.Client

var UserAgent string

func Initialize(userAgent string) {
	UserAgent = userAgent

	HttpClient = &http.Client{
		Transport: &LoggingTransport{},
	}

	Client = ldapi.NewAPIClient(&ldapi.Configuration{
		HTTPClient: HttpClient,
		UserAgent:  UserAgent,
	})
}

func init() {
	SetServer(DefaultServer)
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
