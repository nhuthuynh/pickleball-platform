// Command vulngate is the CI security gate. It reads dependency-scanner
// reports (govulncheck for Go, npm audit for the Vue app), compares them
// against an explicit baseline of accepted findings, and exits non-zero
// when a NEW gating finding appears.
//
// Usage:
//
//	vulngate -baseline security/vuln-baseline.json \
//	         -go build/govulncheck.json \
//	         -npm build/npm-audit.json
//
// Either -go or -npm may be omitted (e.g. when the Go vulnerability
// database is unreachable and that scan was skipped), but at least one
// report is required — running the gate with nothing to check would
// report success while checking nothing, which is worse than not running
// it at all.
//
// All parsing/gating logic lives in tools/vulngate, under test. This file
// is only flag handling and reporting.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/nhuthuynh/white-label/tools/vulngate"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "vulngate: %v\n", err)
		os.Exit(2)
	}
}

func run() error {
	var (
		baselinePath = flag.String("baseline", "security/vuln-baseline.json", "path to the accepted-findings baseline JSON")
		goReport     = flag.String("go", "", "path to `govulncheck -format json` output (optional)")
		npmReport    = flag.String("npm", "", "path to `npm audit --json` output (optional)")
	)
	flag.Parse()

	if *goReport == "" && *npmReport == "" {
		return fmt.Errorf("at least one of -go or -npm is required; refusing to report success without checking anything")
	}

	baseline, err := loadBaseline(*baselinePath)
	if err != nil {
		return err
	}

	var findings []vulngate.Finding

	if *goReport != "" {
		f, err := parseFile(*goReport, vulngate.ParseGovulncheck)
		if err != nil {
			return err
		}
		findings = append(findings, f...)
	}
	if *npmReport != "" {
		f, err := parseFile(*npmReport, vulngate.ParseNPMAudit)
		if err != nil {
			return err
		}
		findings = append(findings, f...)
	}

	res := vulngate.Evaluate(findings, baseline)
	report(res)

	if res.Failed() {
		// Exit 1 = "the gate failed" (actionable), distinct from exit 2 =
		// "the gate could not run" (a broken pipeline, not a vulnerability).
		os.Exit(1)
	}
	return nil
}

func loadBaseline(path string) (vulngate.Baseline, error) {
	f, err := os.Open(path)
	if err != nil {
		return vulngate.Baseline{}, fmt.Errorf("opening baseline %s: %w", path, err)
	}
	defer f.Close()

	b, err := vulngate.LoadBaseline(f)
	if err != nil {
		return vulngate.Baseline{}, fmt.Errorf("%s: %w", path, err)
	}
	return b, nil
}

func parseFile(path string, parse func(io.Reader) ([]vulngate.Finding, error)) ([]vulngate.Finding, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening scanner report %s: %w", path, err)
	}
	defer f.Close()

	findings, err := parse(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return findings, nil
}

func report(res vulngate.Result) {
	section := func(title string, fs []vulngate.Finding) {
		if len(fs) == 0 {
			return
		}
		fmt.Printf("\n%s (%d):\n", title, len(fs))
		for _, f := range fs {
			fmt.Printf("  - %s\n", f)
		}
	}

	section("Below gating threshold (reported, not blocking)", res.Ignored)
	section("Baselined — known and accepted (see security/vuln-baseline.json)", res.Baselined)
	section("NEW gating findings", res.New)

	fmt.Println()
	if res.Failed() {
		fmt.Printf("FAIL: %d new gating finding(s). Fix them, or add an explicit, justified entry to the baseline if they genuinely cannot be fixed here.\n", len(res.New))
		return
	}
	fmt.Printf("PASS: no new gating findings (%d baselined, %d below threshold).\n", len(res.Baselined), len(res.Ignored))
}
