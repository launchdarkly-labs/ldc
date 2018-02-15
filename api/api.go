package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/launchdarkly/foundation/logger"
)

var token string
var server string
var client http.Client

type Link struct {
	Href string `json:"href"`
	Type string `json:"type"`
}
type Links struct {
	Projects []Link `json:"projects"`
}
type LinkList struct {
	Links Links `json:"links"`
}
type ItemList struct {
	Items []json.RawMessage `json:"items"`
}

func init() {
	// TODO remove
	token = ""
	server = "http://localhost/api/v2"
}

func Call(method string, path string, target interface{}) error {
	endpoint, _ := url.Parse(server + path)
	headers := make(map[string][]string)
	headers["Authorization"] = []string{token}
	headers["Content-Type"] = []string{server}
	resp, err := client.Do(&http.Request{
		Method: method,
		URL:    endpoint,
		Header: headers,
	})
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	logger.Info.Println(string(path))
	logger.Info.Println(string(body))
	if err != nil {
		panic(err)
	}
	return json.Unmarshal(body, target)
}

func Name(href string) string {
	segments := strings.Split(href, "/")
	return segments[len(segments)-1]
}

func SetServer(newServer string) {
	server = newServer
}

func SetToken(newToken string) {
	token = newToken
}
