package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/m3yyyyy/driftguard-kubernetes-config-auditor/internal/audit"
)

func Render(format string, result audit.Report) ([]byte, error) {
	switch strings.ToLower(format) {
	case "table":
		return renderTable(result), nil
	case "json":
		output, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode JSON report: %w", err)
		}
		return append(output, '\n'), nil
	case "markdown", "md":
		return renderMarkdown(result), nil
	default:
		return nil, fmt.Errorf("unsupported format %q; use table, json, or markdown", format)
	}
}

func renderTable(result audit.Report) []byte {
	var output bytes.Buffer
	writer := tabwriter.NewWriter(&output, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "STATUS\tRESOURCE\tCHANGED PATHS")
	fmt.Fprintln(writer, "------\t--------\t-------------")
	for _, finding := range result.Findings {
		fmt.Fprintf(writer, "%s\t%s\t%s\n", finding.Status, finding.Resource, strings.Join(finding.Paths, ", "))
	}
	writer.Flush()
	fmt.Fprintf(
		&output,
		"\nsummary: %d total | %d in sync | %d modified | %d missing | %d unexpected | %d errors\n",
		result.Summary.Total,
		result.Summary.InSync,
		result.Summary.Modified,
		result.Summary.Missing,
		result.Summary.Unexpected,
		result.Summary.Errors,
	)
	return output.Bytes()
}

func renderMarkdown(result audit.Report) []byte {
	var output strings.Builder
	output.WriteString("# DriftGuard audit report\n\n")
	fmt.Fprintf(&output, "- Schema: `%s`\n", result.SchemaVersion)
	fmt.Fprintf(&output, "- Context: `%s`\n", escapeMarkdown(result.Context))
	fmt.Fprintf(&output, "- Generated: `%s`\n\n", result.GeneratedAt.Format("2006-01-02T15:04:05Z"))
	output.WriteString("| Status | Resource | Changed paths |\n")
	output.WriteString("|---|---|---|\n")
	for _, finding := range result.Findings {
		fmt.Fprintf(
			&output,
			"| %s | %s | %s |\n",
			escapeMarkdown(string(finding.Status)),
			escapeMarkdown(finding.Resource),
			escapeMarkdown(strings.Join(finding.Paths, ", ")),
		)
	}
	fmt.Fprintf(
		&output,
		"\n**Summary:** %d total; %d in sync; %d modified; %d missing; %d unexpected; %d errors.\n",
		result.Summary.Total,
		result.Summary.InSync,
		result.Summary.Modified,
		result.Summary.Missing,
		result.Summary.Unexpected,
		result.Summary.Errors,
	)
	return []byte(output.String())
}

func escapeMarkdown(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}
