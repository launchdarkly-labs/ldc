package goal_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	ldapi "github.com/launchdarkly/api-client-go"

	"github.com/launchdarkly/ldc/api"
)

const (
	Click    = "click"
	Custom   = "custom"
	PageView = "pageview"
)

var AvailableKinds = []string{Click, Custom, PageView}

// Manually declare the goal type since it isn't part of the v2 api
type Goal struct {
	// Id of the goal
	Id string `json:"_id,omitempty"`

	// Name of the goal
	Name string `json:"name,omitempty"`

	// Description of the goal
	Description string `json:"description,omitempty"`

	// Whether the goal is custom, pageView or click
	Kind string `json:"kind,omitempty"`

	// Whether the goal is being tracked by a flag
	IsActive bool `json:"isActive,omitempty"`

	// A unix epoch time in milliseconds specifying the last modification time of this goal.
	LastModified float32 `json:"lastModified,omitempty"`

	AttachedFeatureCount int `json:"_attachedFeatureCount"`
	AttachedFeatures     []struct {
		Key  string `json:"key"`
		Name string `json:"name"`
		On   bool   `json:"on"`
	} `json:"_attachedFeatures,omitempty"`

	IsDeleteable bool   `json:"_isDeleteable,omitempty"`
	Source       string `json:"_source,omitempty"`
	Version      int    `json:"_version,omitempty"`
}

func GetGoal(key string) (*Goal, error) {
	req, _ := http.NewRequest(http.MethodGet, makeURL("/api/goals/%s", key), nil)
	sdkKey, err := getCurrentSdkKey()
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", sdkKey)

	resp, err := api.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	var goal Goal
	if err := json.Unmarshal(body, &goal); err != nil {
		return nil, err
	}
	return &goal, nil
}

func getCurrentSdkKey() (string, error) {
	env, _, err := api.Client.EnvironmentsApi.GetEnvironment(api.Auth, api.CurrentProject, api.CurrentEnvironment)
	if err != nil {
		return "", err
	}
	return env.ApiKey, nil
}

func GetGoals() ([]Goal, error) {
	req, _ := http.NewRequest(http.MethodGet, makeURL("/api/goals"), nil)
	sdkKey, err := getCurrentSdkKey()
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", sdkKey)

	resp, err := api.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response: %s", resp.Status)
	}

	var goals []Goal
	if err := json.Unmarshal(body, &goals); err != nil {
		return nil, err
	}
	return goals, nil
}

func CreateGoal(goal Goal) (*Goal, error) {
	body, _ := json.Marshal(goal)
	req, _ := http.NewRequest(http.MethodPost, makeURL("/api/goals"), bytes.NewBuffer(body))
	sdkKey, err := getCurrentSdkKey()
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", sdkKey)

	resp, err := api.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected response: %s", resp.Status)
	}

	var newGoal Goal
	if err := json.Unmarshal(respBody, &goal); err != nil {
		return nil, err
	}
	return &newGoal, nil
}

func DeleteGoal(id string) error {
	req, _ := http.NewRequest(http.MethodDelete, makeURL("/api/goals/%s", id), nil)
	sdkKey, err := getCurrentSdkKey()
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", sdkKey)

	resp, err := api.HttpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response: %s", resp.Status)
	}

	return nil
}

func PatchGoal(id string, patchComment ldapi.PatchComment) (*Goal, error) {
	body, _ := json.Marshal(patchComment)
	req, _ := http.NewRequest(http.MethodPatch, makeURL("/api/goals/%s", id), bytes.NewBuffer(body))
	sdkKey, err := getCurrentSdkKey()
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", sdkKey)

	resp, err := api.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response: %s", resp.Status)
	}

	var newGoal Goal
	if err := json.Unmarshal(respBody, &newGoal); err != nil {
		return nil, err
	}
	return &newGoal, nil
}

func makeURL(format string, args ...interface{}) string {
	u, _ := url.Parse(api.CurrentServer)
	u.Path = fmt.Sprintf(format, args...)
	u.RawPath = ""
	return u.String()
}
