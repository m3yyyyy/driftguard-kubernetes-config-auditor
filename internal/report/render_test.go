package report

import (
	"strings"
	"testing"
	"time"

	"github.com/m3yyyyy/driftguard-kubernetes-config-auditor/internal/audit"
)

func TestJSONReportContainsPathsButNoComparedValues(t *testing.T) {
	result := audit.Report{
		SchemaVersion: audit.SchemaVersion,
		GeneratedAt:   time.Unix(0, 0).UTC(),
		Context:       "kind-driftguard",
		Summary:       audit.Summary{Total: 1, Modified: 1},
		Findings: []audit.Finding{{
			Resource: "v1 ConfigMap demo/settings",
			Status:   audit.StatusModified,
			Paths:    []string{"/data/mode"},
			Message:  "one or more desired fields differ",
		}},
	}

	output, err := Render("json", result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "/data/mode") {
		t.Fatalf("expected changed path in report: %s", output)
	}
	if strings.Contains(string(output), "safe") || strings.Contains(string(output), "unsafe") {
		t.Fatalf("report unexpectedly contains compared values: %s", output)
	}
}

func TestRenderRejectsUnknownFormat(t *testing.T) {
	if _, err := Render("xml", audit.Report{}); err == nil {
		t.Fatal("expected unsupported-format error")
	}
}
