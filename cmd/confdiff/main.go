package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ksaegusa/ConfdiffStudio/internal/assertions"
	"github.com/ksaegusa/ConfdiffStudio/internal/bootstrap"
	"github.com/ksaegusa/ConfdiffStudio/internal/check"
	"github.com/ksaegusa/ConfdiffStudio/internal/model"
	"github.com/ksaegusa/ConfdiffStudio/internal/pair"
	"github.com/ksaegusa/ConfdiffStudio/internal/report"
	"github.com/ksaegusa/ConfdiffStudio/internal/structureddiff"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func main() {
	os.Exit(run())
}

func run() int {
	root := &cobra.Command{Use: "confdiff"}
	root.SilenceUsage = true

	var beforeDir string
	var afterDir string
	var glob string
	var assertionPath string
	var outMD string
	var outJSON string
	var outStructuredJSON string
	var outReportJSON string
	var printJSON bool
	var orderMode string
	var ignorePatterns []string
	var replaceRules []string
	var targetPrefixes []string

	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Run assertion checks for before/after config pairs",
		RunE: func(cmd *cobra.Command, args []string) error {
			pairs, err := pair.Resolve(beforeDir, afterDir, glob)
			if err != nil {
				return err
			}

			set, err := assertions.Load(assertionPath)
			if err != nil {
				return err
			}
			if err := assertions.Validate(set); err != nil {
				return err
			}

			r := check.Run(pairs, set)
			fmt.Printf("files_total=%d files_failed=%d\n", r.Summary.FilesTotal, r.Summary.FilesFailed)

			diffProfile := model.DiffProfile{
				OrderMode:      orderMode,
				IgnorePatterns: ignorePatterns,
				TargetPrefixes: targetPrefixes,
			}
			for _, raw := range replaceRules {
				pattern, replacement := parseReplaceRule(raw)
				diffProfile.ReplaceRules = append(diffProfile.ReplaceRules, model.ReplaceRule{
					Pattern:     pattern,
					Replacement: replacement,
				})
			}
			structured, err := structureddiff.BuildFiles(pairs, diffProfile)
			if err != nil {
				return err
			}

			if outJSON != "" {
				if err := report.WriteJSON(outJSON, r); err != nil {
					return err
				}
			}
			if outStructuredJSON != "" {
				if err := writeJSONFile(outStructuredJSON, structured); err != nil {
					return err
				}
			}
			if outReportJSON != "" {
				diffReport := report.BuildDiffReport(structured, r, diffProfile, time.Now().UTC())
				if err := writeJSONFile(outReportJSON, diffReport); err != nil {
					return err
				}
			}
			if outMD != "" {
				if err := report.WriteMarkdown(outMD, r); err != nil {
					return err
				}
			}
			if printJSON {
				if err := writeJSON(os.Stdout, r); err != nil {
					return err
				}
			}

			if r.Summary.FilesFailed > 0 {
				return &exitCodeError{code: 1, err: fmt.Errorf("assertion failures detected")}
			}

			return nil
		},
	}
	checkCmd.Flags().StringVar(&beforeDir, "before-dir", "", "Directory for before configs")
	checkCmd.Flags().StringVar(&afterDir, "after-dir", "", "Directory for after configs")
	checkCmd.Flags().StringVar(&glob, "glob", "*.cfg,*.conf,*.txt,*.log", "File glob")
	checkCmd.Flags().StringVar(&assertionPath, "assertions", "", "Assertion YAML path")
	checkCmd.Flags().StringVar(&outMD, "out-md", "", "Output markdown report path")
	checkCmd.Flags().StringVar(&outJSON, "out-json", "", "Output json report path")
	checkCmd.Flags().StringVar(&outStructuredJSON, "out-structured-json", "", "Output structured diff json path")
	checkCmd.Flags().StringVar(&outReportJSON, "out-report-json", "", "Output diff report json path")
	checkCmd.Flags().BoolVar(&printJSON, "print-json", false, "Print report JSON to stdout")
	checkCmd.Flags().StringVar(&orderMode, "order-mode", "lenient", "Diff order mode: lenient or strict")
	checkCmd.Flags().StringArrayVar(&ignorePatterns, "ignore-pattern", nil, "Ignore regex for diff preprocessing")
	checkCmd.Flags().StringArrayVar(&replaceRules, "replace-rule", nil, "Replace rule in the form pattern=>replacement")
	checkCmd.Flags().StringArrayVar(&targetPrefixes, "target-prefix", nil, "Target line prefix for scoped diff")
	_ = checkCmd.MarkFlagRequired("before-dir")
	_ = checkCmd.MarkFlagRequired("after-dir")
	_ = checkCmd.MarkFlagRequired("assertions")

	var bootstrapOut string
	var bootstrapStdout bool
	bootstrapCmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Generate starter assertion YAML from before/after pairs",
		RunE: func(cmd *cobra.Command, args []string) error {
			pairs, err := pair.Resolve(beforeDir, afterDir, glob)
			if err != nil {
				return err
			}
			set := bootstrap.Generate(pairs)
			if bootstrapStdout {
				data, err := yaml.Marshal(set)
				if err != nil {
					return err
				}
				fmt.Print(string(data))
				return nil
			}
			if err := assertions.Save(bootstrapOut, set); err != nil {
				return err
			}
			fmt.Printf("generated %d rules: %s\n", len(set.Rules), bootstrapOut)
			return nil
		},
	}
	bootstrapCmd.Flags().StringVar(&beforeDir, "before-dir", "", "Directory for before configs")
	bootstrapCmd.Flags().StringVar(&afterDir, "after-dir", "", "Directory for after configs")
	bootstrapCmd.Flags().StringVar(&glob, "glob", "*.cfg,*.conf,*.txt,*.log", "File glob")
	bootstrapCmd.Flags().StringVar(&bootstrapOut, "out", "assertions.generated.yaml", "Output assertion YAML path")
	bootstrapCmd.Flags().BoolVar(&bootstrapStdout, "stdout", false, "Print generated YAML to stdout")
	_ = bootstrapCmd.MarkFlagRequired("before-dir")
	_ = bootstrapCmd.MarkFlagRequired("after-dir")

	var validatePath string
	var validateJSON bool
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate assertion YAML syntax and rule schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			set, err := assertions.Load(validatePath)
			if err != nil {
				if validateJSON {
					_ = writeJSON(os.Stdout, map[string]any{"valid": false, "error": err.Error()})
					return &exitCodeError{code: 1, err: err}
				}
				return err
			}
			if err := assertions.Validate(set); err != nil {
				if validateJSON {
					_ = writeJSON(os.Stdout, map[string]any{"valid": false, "error": err.Error()})
					return &exitCodeError{code: 1, err: err}
				}
				return err
			}
			if validateJSON {
				return writeJSON(os.Stdout, map[string]any{"valid": true})
			}
			fmt.Println("assertions are valid")
			return nil
		},
	}
	validateCmd.Flags().StringVar(&validatePath, "assertions", "", "Assertion YAML path")
	validateCmd.Flags().BoolVar(&validateJSON, "json", false, "Print validation result as JSON")
	_ = validateCmd.MarkFlagRequired("assertions")

	root.AddCommand(checkCmd, bootstrapCmd, validateCmd)

	if err := root.Execute(); err != nil {
		var codeErr *exitCodeError
		if ok := asExitCodeError(err, &codeErr); ok {
			fmt.Fprintln(os.Stderr, codeErr.err)
			return codeErr.code
		}
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	return 0
}

func writeJSON(w *os.File, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeJSONFile(path string, v any) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return writeJSON(file, v)
}

func parseReplaceRule(raw string) (string, string) {
	if idx := strings.Index(raw, "=>"); idx >= 0 {
		return strings.TrimSpace(raw[:idx]), strings.TrimSpace(raw[idx+2:])
	}
	if idx := strings.Index(raw, "="); idx >= 0 {
		return strings.TrimSpace(raw[:idx]), strings.TrimSpace(raw[idx+1:])
	}
	return strings.TrimSpace(raw), ""
}

type exitCodeError struct {
	code int
	err  error
}

func (e *exitCodeError) Error() string {
	return e.err.Error()
}

func asExitCodeError(err error, target **exitCodeError) bool {
	typed, ok := err.(*exitCodeError)
	if !ok {
		return false
	}
	*target = typed
	return true
}
