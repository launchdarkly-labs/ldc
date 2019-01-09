/*
 * LaunchDarkly REST API
 *
 * Build custom integrations with the LaunchDarkly REST API
 *
 * API version: 2.0.13
 * Contact: support@launchdarkly.com
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package ldapi

type EnvironmentPost struct {

	// The name of the new environment.
	Name string `json:"name"`

	// A project-unique key for the new environment.
	Key string `json:"key"`

	// A color swatch (as an RGB hex value with no leading '#', e.g. C8C8C8).
	Color string `json:"color"`

	// The default TTL for the new environment.
	DefaultTtl float32 `json:"defaultTtl,omitempty"`
}
