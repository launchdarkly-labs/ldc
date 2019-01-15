// Package goalapi provides goals via the v1 api.  goals are not yet included in the v2 api
package goalapi

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
	// Click represents a click goal
	Click = "click"
	// Custom represents a custom event goal
	Custom = "custom"
	// PageView indicates a page view goal
	PageView = "pageview"
)

// Kinds are all the kinds that we can use for a goal
var Kinds = []string{Click, Custom, PageView}

// URLMatcherBase includes the kind of a url matcher in url matcher definitions
type URLMatcherBase struct {
	Kind string `json:"kind"`
}

// URLMatcherCanonical describes a canonical url matcher
type URLMatcherCanonical struct {
	URLMatcherBase `json:",inline"`
	URL            string `json:"url"`
}

// URLMatcherExact describes an exact url matcher
type URLMatcherExact struct {
	URLMatcherBase `json:",inline"`
	URL            string `json:"url"`
}

// URLMatcherSubstring describes a substring url matcher
type URLMatcherSubstring struct {
	URLMatcherBase `json:",inline"`
	Substring      string `json:"substring"`
}

// URLMatcherRegex describes a regex url matcher
type URLMatcherRegex struct {
	URLMatcherBase `json:",inline"`
	Pattern        string `json:"pattern"`
}

// GoalURLMatchers describes the url matchers for a goal
type GoalURLMatchers struct {
	ExactURLs     []URLMatcherExact     `json:"exactUrls,omitempty"`
	CanonicalURLs []URLMatcherCanonical `json:"canonicalUrls,omitempty"`
	RegexURLs     []URLMatcherRegex     `json:"regexUrls,omitempty"`
	SubstringURLs []URLMatcherSubstring `json:"substringUrls,omitempty"`
}

// Goal describes the goal type.
type Goal struct {
	// ID of the goal
	ID string `json:"_id,omitempty"`

	// Name of the goal
	Name string `json:"name,omitempty"`

	// Description of the goal
	Description string `json:"description,omitempty"`

	// Kind tells whether the goal is custom, pageView or click
	Kind string `json:"kind,omitempty"`

	// Key for custom goals
	Key *string `json:"key,omitempty"`

	// IsActive indicates whether the goal is being tracked by a flag
	IsActive bool `json:"isActive,omitempty"`

	// LastModified is a unix epoch time in milliseconds specifying the last modification time of this goal.
	LastModified float32 `json:"lastModified,omitempty"`

	// AttachedFeatureCount is the number of attached goals
	AttachedFeatureCount int `json:"_attachedFeatureCount,omitempty"`

	// URLs are the url matchers attached to the goal
	URLs []GoalURLMatchers `json:"urls,omitempty"`

	// AttachedFeatures describes the flags attached to this goal.  This is on the individual goal view.
	AttachedFeatures []struct {
		Key  string `json:"key"`
		Name string `json:"name"`
		On   bool   `json:"on"`
	} `json:"_attachedFeatures,omitempty"`

	// IsDeleteable indicates if the goal can be deleted
	IsDeleteable bool `json:"_isDeleteable,omitempty"`

	// Source is the source of the goal
	Source *struct {
		// Name is the name of the source
		Name string `json:"name"`
	} `json:"_source,omitempty"`

	// Version is the version of the goal
	Version int `json:"_version,omitempty"`
}

// ExperimentResults holds the results of an experiment for a particular goal/flag combo
type ExperimentResults struct {
	Change          float64   `json:"change"`
	ConfidenceScore float64   `json:"confidenceScore"`
	ZScore          float64   `json:"z_score"`
	Control         Variation `json:"control"`
	Experiment      Variation `json:"experiment"`
}

// Variation holds data about each result
type Variation struct {
	Conversions        int     `json:"conversions"`
	Impressions        int     `json:"impressions"`
	ConversionRate     float64 `json:"conversionRate"`
	StandardError      float64 `json:"standardError"`
	ConfidenceInterval float64 `json:"confidenceInterval"`
}

// GetGoal returns the goal with a given id
func GetGoal(id string) (*Goal, error) {
	req, _ := http.NewRequest(http.MethodGet, makeURL("/api/goals/%s", id), nil)
	sdkKey, err := getCurrentSdkKey()
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", sdkKey)

	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	_ = resp.Body.Close()

	var goal Goal
	if err := json.Unmarshal(body, &goal); err != nil {
		return nil, err
	}
	return &goal, nil
}

// GetExperimentResults returns the experiment results for a specific flag and goal
func GetExperimentResults(goalID string, flagKey string) (*ExperimentResults, error) {
	req, _ := http.NewRequest(http.MethodGet, makeURL("/api/features/%s/goals/%s/results", flagKey, goalID), nil)
	sdkKey, err := getCurrentSdkKey()
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", sdkKey)
	req.Header.Add("Content-Type", "application/json")

	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	_ = resp.Body.Close()

	var experimentResults ExperimentResults
	if err := json.Unmarshal(body, &experimentResults); err != nil {
		return nil, err
	}
	return &experimentResults, nil
}

func getCurrentSdkKey() (string, error) {
	env, _, err := api.Client.EnvironmentsApi.GetEnvironment(api.Auth, api.CurrentProject, api.CurrentEnvironment)
	if err != nil {
		return "", err
	}
	return env.ApiKey, nil
}

// GetGoals returns all goals for the current environment
func GetGoals() ([]Goal, error) {
	req, _ := http.NewRequest(http.MethodGet, makeURL("/api/goals"), nil)
	sdkKey, err := getCurrentSdkKey()
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", sdkKey)

	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response: %s", resp.Status)
	}

	var respData struct {
		Items []Goal
	}

	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, fmt.Errorf("Unable to unmarshal: %s: %s", body, err)
	}

	return respData.Items, nil
}

// CreateGoal creates a goal in the current environment
func CreateGoal(goal Goal) (*Goal, error) {
	body, _ := json.Marshal(goal)
	req, _ := http.NewRequest(http.MethodPost, makeURL("/api/goals"), bytes.NewBuffer(body))
	sdkKey, err := getCurrentSdkKey()
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", sdkKey)
	req.Header.Add("Content-Type", "application/json")

	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected response: %s", resp.Status)
	}

	var newGoal Goal
	if err := json.Unmarshal(respBody, &goal); err != nil {
		return nil, err
	}
	return &newGoal, nil
}

// DeleteGoal deletes a goal in the current environment
func DeleteGoal(id string) error {
	req, _ := http.NewRequest(http.MethodDelete, makeURL("/api/goals/%s", id), nil)
	sdkKey, err := getCurrentSdkKey()
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", sdkKey)

	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected response: %s", resp.Status)
	}

	return nil
}

// PatchGoal patches a goal in the current environment
func PatchGoal(id string, patchComment ldapi.PatchComment) (*Goal, error) {
	body, _ := json.Marshal(patchComment)
	req, _ := http.NewRequest(http.MethodPatch, makeURL("/api/goals/%s", id), bytes.NewBuffer(body))
	sdkKey, err := getCurrentSdkKey()
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", sdkKey)

	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	_ = resp.Body.Close()

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
