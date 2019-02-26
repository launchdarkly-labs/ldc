package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	ldapi "github.com/launchdarkly/api-client-go"
)

const defaultServerURL = "https://app.launchdarkly.com"

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
}

// GetClient returns a client for the given server
func GetClient(server string) (*ldapi.APIClient, error) {
	if server == "" {
		server = defaultServerURL
	}
	url, err := url.Parse(server)
	if err != nil {
		return nil, fmt.Errorf("unable to parser server: %s", err)
	}
	url.Path = "/api/v2"
	url.RawPath = ""
	return ldapi.NewAPIClient(&ldapi.Configuration{
		BasePath:   url.String(),
		HTTPClient: HTTPClient,
		UserAgent:  UserAgent,
	}), nil
}

// GetAuthCtx returns a context that can be used to access the api
func GetAuthCtx(token string) context.Context {
	return context.WithValue(context.Background(), ldapi.ContextAPIKey, ldapi.APIKey{
		Key: token,
	})
}
