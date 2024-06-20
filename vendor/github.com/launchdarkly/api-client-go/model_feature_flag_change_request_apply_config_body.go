/*
 * LaunchDarkly REST API
 *
 * Build custom integrations with the LaunchDarkly REST API
 *
 * API version: 3.10.0
 * Contact: support@launchdarkly.com
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package ldapi

type FeatureFlagChangeRequestApplyConfigBody struct {
	// comment will be included in audit log item for change.
	Comment string `json:"comment,omitempty"`
}
