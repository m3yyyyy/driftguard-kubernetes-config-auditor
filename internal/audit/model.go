package audit

import "time"

const SchemaVersion = "driftguard/v1"

type Status string

const (
	StatusInSync     Status = "in_sync"
	StatusModified   Status = "modified"
	StatusMissing    Status = "missing"
	StatusUnexpected Status = "unexpected"
	StatusError      Status = "error"
)

type Finding struct {
	Resource string   `json:"resource"`
	Status   Status   `json:"status"`
	Paths    []string `json:"paths,omitempty"`
	Message  string   `json:"message"`
}

type Summary struct {
	Total      int `json:"total"`
	InSync     int `json:"in_sync"`
	Modified   int `json:"modified"`
	Missing    int `json:"missing"`
	Unexpected int `json:"unexpected"`
	Errors     int `json:"errors"`
}

func (summary Summary) HasDrift() bool {
	return summary.Modified+summary.Missing+summary.Unexpected > 0
}

type Report struct {
	SchemaVersion string    `json:"schema_version"`
	GeneratedAt   time.Time `json:"generated_at"`
	Context       string    `json:"context"`
	Summary       Summary   `json:"summary"`
	Findings      []Finding `json:"findings"`
}
