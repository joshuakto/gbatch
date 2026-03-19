package migrate

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// UGERDirective represents a parsed UGER #$ directive.
type UGERDirective struct {
	Flag  string
	Value string
	Line  int
}

// Result holds the migration output.
type Result struct {
	CPUs       int
	Mem        string
	Queue      string
	JobName    string
	Script     string
	Warnings   []string
	Directives []UGERDirective
}

var directiveRe = regexp.MustCompile(`^#\$\s+(.+)`)

// ParseFile reads a UGER submission script and extracts #$ directives.
func ParseFile(path string) (*Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %w", path, err)
	}
	defer f.Close()

	result := &Result{Script: path}
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		matches := directiveRe.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		flagStr := strings.TrimSpace(matches[1])
		parseDirective(flagStr, lineNum, result)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	if len(result.Directives) == 0 {
		return nil, fmt.Errorf("no UGER directives (#$) found in %s", path)
	}

	return result, nil
}

func parseDirective(flagStr string, line int, result *Result) {
	parts := strings.Fields(flagStr)
	if len(parts) == 0 {
		return
	}

	i := 0
	for i < len(parts) {
		flag := parts[i]
		dir := UGERDirective{Flag: flag, Line: line}

		switch flag {
		case "-l":
			// Resource request: -l h_vmem=32G, -l h_rt=24:00:00, etc.
			if i+1 < len(parts) {
				i++
				dir.Value = parts[i]
				parseResourceList(parts[i], result)
			}
		case "-pe":
			// Parallel environment: -pe smp 8
			if i+2 < len(parts) {
				i += 2
				dir.Value = parts[i-1] + " " + parts[i]
				if n, err := strconv.Atoi(parts[i]); err == nil {
					result.CPUs = n
				}
			}
		case "-q":
			// Queue: -q long
			if i+1 < len(parts) {
				i++
				dir.Value = parts[i]
				result.Queue = parts[i]
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("line %d: -q %s has no direct GCP equivalent (ignored)", line, parts[i]))
			}
		case "-N":
			// Job name: -N my_job
			if i+1 < len(parts) {
				i++
				dir.Value = parts[i]
				result.JobName = parts[i]
			}
		case "-cwd", "-V", "-j", "-b", "-r", "-notify":
			// Common flags with no value or no GCP equivalent
			dir.Value = ""
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("line %d: %s not supported in GCP Batch (ignored)", line, flag))
		case "-o", "-e":
			// Output/error file paths
			if i+1 < len(parts) {
				i++
				dir.Value = parts[i]
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("line %d: %s %s — logs handled by Cloud Logging instead", line, flag, parts[i]))
			}
		default:
			dir.Value = ""
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("line %d: unknown flag %s (ignored)", line, flag))
		}

		result.Directives = append(result.Directives, dir)
		i++
	}
}

func parseResourceList(val string, result *Result) {
	// Resources are comma-separated: h_vmem=32G,h_rt=24:00:00
	for _, res := range strings.Split(val, ",") {
		kv := strings.SplitN(res, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key, value := kv[0], kv[1]
		switch key {
		case "h_vmem", "mem_free":
			result.Mem = value
		case "h_rt":
			// Runtime limit — no direct GCP equivalent, warn
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("%s=%s — GCP Batch has no runtime limit (ignored)", key, value))
		}
	}
}

// ToGbatchCommand converts the parsed result to a gbatch submit command.
func (r *Result) ToGbatchCommand() string {
	var parts []string
	parts = append(parts, "gbatch submit")

	if r.CPUs > 0 {
		parts = append(parts, fmt.Sprintf("--cpus %d", r.CPUs))
	}
	if r.Mem != "" {
		parts = append(parts, fmt.Sprintf("--mem %s", r.Mem))
	}
	if r.JobName != "" {
		parts = append(parts, fmt.Sprintf("--name %s", r.JobName))
	}

	parts = append(parts, r.Script)
	return strings.Join(parts, " ")
}
