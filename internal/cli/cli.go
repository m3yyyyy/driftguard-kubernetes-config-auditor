package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	auditpkg "github.com/m3yyyyy/driftguard-kubernetes-config-auditor/internal/audit"
	"github.com/m3yyyyy/driftguard-kubernetes-config-auditor/internal/kube"
	"github.com/m3yyyyy/driftguard-kubernetes-config-auditor/internal/manifest"
	"github.com/m3yyyyy/driftguard-kubernetes-config-auditor/internal/report"
)

var Version = "0.1.0-dev"

const (
	exitOK       = 0
	exitDrift    = 2
	exitUsage    = 64
	exitInternal = 70
)

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return exitUsage
	}

	switch args[0] {
	case "audit":
		return runAudit(args[1:], stdout, stderr)
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintf(stdout, "driftguard %s\n", Version)
		return exitOK
	case "help", "-h", "--help":
		printUsage(stdout)
		return exitOK
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printUsage(stderr)
		return exitUsage
	}
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("path", "", "manifest file or directory")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitUsage
	}
	if *path == "" || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "validate requires --path and no positional arguments")
		return exitUsage
	}

	objects, err := manifest.Load(*path)
	if err != nil {
		fmt.Fprintln(stderr, "validation failed:", err)
		return exitUsage
	}
	fmt.Fprintf(stdout, "valid: %d managed Kubernetes object(s)\n", len(objects))
	return exitOK
}

func runAudit(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("audit", flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("path", "", "manifest file or directory")
	contextName := flags.String("context", "", "kubeconfig context; defaults to the current context")
	defaultNamespace := flags.String("default-namespace", "default", "namespace for namespaced manifests that omit metadata.namespace")
	format := flags.String("format", "table", "output format: table, json, or markdown")
	outputPath := flags.String("output", "", "optional report file")
	timeout := flags.Duration("timeout", 30*time.Second, "overall Kubernetes read timeout")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitUsage
	}
	if *path == "" || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "audit requires --path and no positional arguments")
		return exitUsage
	}
	if *timeout < time.Second || *timeout > 5*time.Minute {
		fmt.Fprintln(stderr, "--timeout must be between 1s and 5m")
		return exitUsage
	}

	objects, err := manifest.Load(*path)
	if err != nil {
		fmt.Fprintln(stderr, "load desired manifests:", err)
		return exitUsage
	}
	reader, err := kube.New(*contextName)
	if err != nil {
		fmt.Fprintln(stderr, "connect to Kubernetes:", err)
		return exitInternal
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	result := auditpkg.Run(ctx, objects, reader, *defaultNamespace)
	rendered, err := report.Render(*format, result)
	if err != nil {
		fmt.Fprintln(stderr, "render report:", err)
		return exitUsage
	}

	if *outputPath == "" {
		if _, err = stdout.Write(rendered); err != nil {
			fmt.Fprintln(stderr, "write report:", err)
			return exitInternal
		}
	} else {
		cleanPath := filepath.Clean(*outputPath)
		if parent := filepath.Dir(cleanPath); parent != "." {
			if err = os.MkdirAll(parent, 0o755); err != nil {
				fmt.Fprintln(stderr, "create report directory:", err)
				return exitInternal
			}
		}
		if err = os.WriteFile(cleanPath, rendered, 0o600); err != nil {
			fmt.Fprintln(stderr, "write report:", err)
			return exitInternal
		}
		fmt.Fprintf(stdout, "report written: %s\n", cleanPath)
	}

	if result.Summary.Errors > 0 {
		return exitInternal
	}
	if result.Summary.HasDrift() {
		return exitDrift
	}
	return exitOK
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, "DriftGuard - read-only Kubernetes configuration drift auditor")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Usage:")
	fmt.Fprintln(writer, "  driftguard validate --path <file-or-directory>")
	fmt.Fprintln(writer, "  driftguard audit --path <file-or-directory> [--context name] [--format table|json|markdown]")
	fmt.Fprintln(writer, "  driftguard version")
}
