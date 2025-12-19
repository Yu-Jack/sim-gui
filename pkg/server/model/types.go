package model

import "time"

type Workspace struct {
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	CreatedAt   time.Time `json:"createdAt"`
	Versions    []Version `json:"versions"`
}

type VersionType string

const (
	VersionTypeSupportBundle VersionType = "support-bundle"
	VersionTypeRuntime       VersionType = "runtime"
)

type Version struct {
	ID                string      `json:"id"`   // e.g., v1, v2
	Name              string      `json:"name"` // User provided name or filename
	Type              VersionType `json:"type"` // "support-bundle" or "runtime"
	CreatedAt         time.Time   `json:"createdAt"`
	Path              string      `json:"path"`           // Path to the extracted data
	BundlePath        string      `json:"bundlePath"`     // Path to the original zip file
	KubeconfigPath    string      `json:"kubeconfigPath"` // Path to the kubeconfig file
	SupportBundleName string      `json:"supportBundleName"`
	Ready             bool        `json:"ready"`
}
