package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	benchmarkLinePattern = regexp.MustCompile(`^(Benchmark(?:SearchIndex|TreeRebuild)/\S+)\s+\d+\s+(\d+)\s+ns/op`)
	cpuSuffixPattern     = regexp.MustCompile(`-\d+$`)
	expectedBenchmarks   = []string{
		"BenchmarkSearchIndex/small/cold-build",
		"BenchmarkSearchIndex/small/warm-query",
		"BenchmarkSearchIndex/large/cold-build",
		"BenchmarkSearchIndex/large/warm-query",
		"BenchmarkTreeRebuild/medium/cold-cache-rebuild",
		"BenchmarkTreeRebuild/medium/warm-cache-rebuild",
		"BenchmarkTreeRebuild/large/cold-cache-rebuild",
		"BenchmarkTreeRebuild/large/warm-cache-rebuild",
	}
)

type comparisonRow struct {
	name       string
	baselineNs float64
	currentNs  float64
	deltaPct   float64
	pass       bool
}

func main() {
	baselinePath := flag.String("baseline", "", "path to baseline benchmark output")
	currentPath := flag.String("current", "", "path to current benchmark output")
	maxRegressionPct := flag.Float64("max-regression-pct", 20, "maximum allowed regression percent before failing")
	flag.Parse()

	if *baselinePath == "" || *currentPath == "" {
		fatalf("both -baseline and -current are required")
	}
	if *maxRegressionPct < 0 {
		fatalf("-max-regression-pct must be non-negative")
	}

	baseline, err := parseBenchmarkOutput(*baselinePath)
	if err != nil {
		fatalf("parse baseline: %v", err)
	}
	current, err := parseBenchmarkOutput(*currentPath)
	if err != nil {
		fatalf("parse current: %v", err)
	}

	rows, err := compareBenchmarks(baseline, current, *maxRegressionPct)
	if err != nil {
		fatalf("compare benchmarks: %v", err)
	}

	writeMarkdownReport(rows, *maxRegressionPct, os.Stdout)
	if stepSummary := os.Getenv("GITHUB_STEP_SUMMARY"); stepSummary != "" {
		f, err := os.OpenFile(stepSummary, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
		if err != nil {
			fatalf("open step summary: %v", err)
		}
		defer f.Close()
		writeMarkdownReport(rows, *maxRegressionPct, f)
	}

	for _, row := range rows {
		if !row.pass {
			os.Exit(1)
		}
	}
}

func parseBenchmarkOutput(path string) (map[string]float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer file.Close()

	results := make(map[string]float64, len(expectedBenchmarks))
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := benchmarkLinePattern.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}

		name := cpuSuffixPattern.ReplaceAllString(matches[1], "")
		nsPerOp, err := strconv.ParseFloat(matches[2], 64)
		if err != nil {
			return nil, fmt.Errorf("parse ns/op for %q: %w", name, err)
		}
		results[name] = nsPerOp
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %q: %w", path, err)
	}

	if len(results) == 0 {
		return nil, errors.New("no benchmark results found for expected suites")
	}
	return results, nil
}

func compareBenchmarks(baseline, current map[string]float64, maxRegressionPct float64) ([]comparisonRow, error) {
	for _, name := range expectedBenchmarks {
		if _, ok := current[name]; !ok {
			return nil, fmt.Errorf("missing current benchmark %q", name)
		}
	}

	rows := make([]comparisonRow, 0, len(expectedBenchmarks))
	for _, name := range expectedBenchmarks {
		base, ok := baseline[name]
		if !ok {
			// A newly added benchmark has no baseline yet. Treat current as
			// baseline-equivalent so this change can land; future runs will
			// compare real baseline vs current values.
			base = current[name]
		}
		if base <= 0 {
			return nil, fmt.Errorf("non-positive baseline ns/op for %q", name)
		}
		curr := current[name]
		delta := ((curr - base) / base) * 100
		rows = append(rows, comparisonRow{
			name:       name,
			baselineNs: base,
			currentNs:  curr,
			deltaPct:   delta,
			pass:       delta <= maxRegressionPct,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].name < rows[j].name
	})
	return rows, nil
}

func writeMarkdownReport(rows []comparisonRow, maxRegressionPct float64, out *os.File) {
	fmt.Fprintf(out, "## Benchmark Comparison\n\n")
	fmt.Fprintf(out, "Allowed regression threshold: %.2f%%\n\n", maxRegressionPct)
	fmt.Fprintf(out, "| Benchmark | Baseline ns/op | Current ns/op | Delta | Result |\n")
	fmt.Fprintf(out, "|---|---:|---:|---:|---|\n")
	for _, row := range rows {
		result := "PASS"
		if !row.pass {
			result = "FAIL"
		}
		delta := 0.0
		if !math.IsNaN(row.deltaPct) && !math.IsInf(row.deltaPct, 0) {
			delta = row.deltaPct
		}
		fmt.Fprintf(out, "| %s | %.0f | %.0f | %+0.2f%% | %s |\n", row.name, row.baselineNs, row.currentNs, delta, result)
	}
	fmt.Fprintln(out)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}
