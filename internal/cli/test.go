package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jiejie-dev/funny/v2/internal/testrunner"
)

// Test runs funny test files and prints a summary.
func Test(path string, verbose, jsonOut bool) error {
	report, err := testrunner.Run(testrunner.Options{Path: path, Verbose: verbose})
	if err != nil {
		return err
	}
	if jsonOut {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	} else if !verbose {
		for _, t := range report.Tests {
			status := "PASS"
			if !t.Passed {
				status = "FAIL"
			}
			fmt.Fprintf(os.Stdout, "%s  %s  %s\n", status, t.File, t.Name)
			if t.Error != "" {
				fmt.Fprintf(os.Stdout, "       %s\n", t.Error)
			}
		}
	}
	fmt.Fprintf(os.Stdout, "\n%d passed, %d failed\n", report.Passed, report.Failed)
	if report.Failed > 0 {
		return fmt.Errorf("%d test(s) failed", report.Failed)
	}
	return nil
}
