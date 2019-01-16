package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	ldapi "github.com/launchdarkly/api-client-go"
)

// Auth is the authorization context used by the ali client
var Auth context.Context

// Client is the api client
var Client *ldapi.APIClient

const defaultServer = "https://app.launchdarkly.com/api/v2"

// CurrentToken is the api token
var CurrentToken string

// CurrentServer is the url of the api to use
var CurrentServer string

// CurrentProject is the project to use
var CurrentProject = "default"

// CurrentEnvironment is the environment to use
var CurrentEnvironment = "production"

// HTTPClient is an underlying http client with logging transport
var HTTPClient *http.Client

// UserAgent is the current user agent for this version of the command
var UserAgent string

// Debug turns on debugging of http requests
var Debug bool

type loggingTransport struct{}

func (lt *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if Debug {
		if req.Body != nil {
			body, _ := ioutil.ReadAll(req.Body)
			fmt.Printf("body: %s\n", string(body))
			req.GetBody = func() (io.ReadCloser, error) {
				return ioutil.NopCloser(bytes.NewBuffer(body)), nil
			}
		}
	}

	resp, err := http.DefaultTransport.RoundTrip(req)

	if Debug && req.Body != nil && err != nil {
		body, err := ioutil.ReadAll(req.Body)
		if err == nil && body != nil {
			_ = resp.Body.Close()
			fmt.Printf("response: %s\n", string(body))
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		}
	}
	return resp, err
}

// Initialize sets up api for use with a given user agent string
func Initialize(userAgent string) {
	UserAgent = userAgent

	HTTPClient = &http.Client{
		Transport: &loggingTransport{},
	}

	Client = ldapi.NewAPIClient(&ldapi.Configuration{
		HTTPClient: HTTPClient,
		UserAgent:  UserAgent,
	})
}

func init() {
	SetServer(defaultServer)
}

// SetServer sets the server url to use
func SetServer(newServer string) {
	CurrentServer = newServer
	Client = ldapi.NewAPIClient(&ldapi.Configuration{
		BasePath: newServer,
		HTTPClient: &http.Client{
			Transport: &loggingTransport{},
		},
		UserAgent: "ldc/0.0.1/go",
	})
}

// SetToken sets the authorization token
func SetToken(newToken string) {
	CurrentToken = newToken
	Auth = context.WithValue(context.Background(), ldapi.ContextAPIKey, ldapi.APIKey{
		Key: newToken,
	})
}
