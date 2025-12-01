package model

import "time"

type Workspace struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	Versions  []Version `json:"versions"`
}

type Version struct {
	ID                string    `json:"id"`   // e.g., v1, v2
	Name              string    `json:"name"` // User provided name or filename
	CreatedAt         time.Time `json:"createdAt"`
	Path              string    `json:"path"`       // Path to the extracted data
	BundlePath        string    `json:"bundlePath"` // Path to the original zip file
	SupportBundleName string    `json:"supportBundleName"`
	Ready             bool      `json:"ready"`
}
