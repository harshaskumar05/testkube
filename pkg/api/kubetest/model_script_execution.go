/*
 * Kubetest
 *
 * Efficient testing of k8s applications mandates a k8s native approach to test mgmt/definition/execution - kubetest provides a “quality control plane” that natively integrates testing activities into k8s development and operational workflows
 *
 * API version: 0.0.1
 * Contact: api@kubetest.io
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package kubetest

// API server script execution
type ScriptExecution struct {
	// execution id
	Id string `json:"id,omitempty"`
	// unique script name (CRD Script name)
	ScriptName string `json:"script-name,omitempty"`
	// script type e.g. postman/collection
	ScriptType string `json:"script-type,omitempty"`
	// execution name
	Name string `json:"name,omitempty"`
	// execution params passed to executor
	Params map[string]string `json:"params,omitempty"`
	Execution *Execution `json:"execution,omitempty"`
}
