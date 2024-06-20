/*
 * LaunchDarkly REST API
 *
 * Build custom integrations with the LaunchDarkly REST API
 *
 * API version: 5.3.0
 * Contact: support@launchdarkly.com
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package ldapi

type StreamUsageSeries struct {
	// A key corresponding to a time series data point.
	Var0 int64 `json:"0,omitempty"`
	// A unix epoch time in milliseconds specifying the creation time of this flag.
	Time int64 `json:"time,omitempty"`
}