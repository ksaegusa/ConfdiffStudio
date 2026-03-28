package studio

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ksaegusa/ConfdiffStudio/internal/assertions"
	"github.com/ksaegusa/ConfdiffStudio/internal/check"
	"github.com/ksaegusa/ConfdiffStudio/internal/license"
	"github.com/ksaegusa/ConfdiffStudio/internal/model"
	"github.com/ksaegusa/ConfdiffStudio/internal/report"
	"github.com/ksaegusa/ConfdiffStudio/internal/structureddiff"
	"gopkg.in/yaml.v3"
)

const defaultRuleset = `version: 1
defaults:
  match_mode: regex
rules:
  - id: no-any-any
    type: line_absent
    severity: high
    pattern: '(?i)(permit|allow)\s+\w+\s+any\s+any'
  - id: keep-deny-drop
    type: diff_forbid_removed
    severity: high
    pattern: '(?i)\b(deny|drop)\b'
  - id: block-default-route-open
    type: diff_forbid_removed
    severity: medium
    pattern: '(?i)0\.0\.0\.0/0'
`

const maxLicenseEnvelopeBytes = 64 * 1024

type Options struct {
	LicenseFile   string
	PublicKeyFile string
}

type checkRequest struct {
	Pairs         []pairInput  `json:"pairs"`
	AssertionYAML string       `json:"assertionYaml"`
	WantMarkdown  bool         `json:"wantMarkdown"`
	WantReport    bool         `json:"wantReport"`
	Profile       checkProfile `json:"profile"`
}

type pairInput struct {
	Name   string `json:"name"`
	Before string `json:"before"`
	After  string `json:"after"`
}

type checkProfile struct {
	OrderMode      string        `json:"orderMode"`
	IgnorePatterns []string      `json:"ignorePatterns"`
	ReplaceRules   []replaceRule `json:"replaceRules"`
	TargetPrefixes []string      `json:"targetPrefixes"`
}

type replaceRule struct {
	Pattern     string `json:"pattern"`
	Replacement string `json:"replacement"`
}

type checkResponse struct {
	OK             bool                       `json:"ok"`
	ExitCode       int                        `json:"exitCode"`
	Report         model.CheckReport          `json:"report"`
	StructuredDiff []model.StructuredDiffFile `json:"structuredDiff"`
	DiffReport     *model.DiffReportPayload   `json:"diffReport,omitempty"`
	Markdown       string                     `json:"markdown"`
	Stdout         string                     `json:"stdout"`
	Stderr         string                     `json:"stderr"`
}

type applyLicenseRequest struct {
	LicenseEnvelope string `json:"licenseEnvelope"`
}

func NewHandler(staticFS embed.FS, opts Options) (http.Handler, error) {
	webFS, err := fs.Sub(staticFS, "web/dist")
	if err != nil {
		return nil, fmt.Errorf("open embedded web assets: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleCheck(w, r, opts)
	})
	mux.HandleFunc("/api/license/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleLicenseStatus(w, opts)
	})
	mux.HandleFunc("/api/license/apply", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleLicenseApply(w, r, opts)
	})
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
	mux.Handle("/", spaHandler(webFS))

	return withAccessLog(mux), nil
}

func handleCheck(w http.ResponseWriter, r *http.Request, opts Options) {
	var req checkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if len(req.Pairs) == 0 {
		writeError(w, http.StatusBadRequest, "pairs are required")
		return
	}

	status := resolveLicenseStatus(opts)
	policy := license.PolicyForStatus(status)
	violation := license.ValidateCheckInput(policy, license.CheckInput{
		Profile: license.CheckProfile{
			OrderMode:      req.Profile.OrderMode,
			IgnorePatterns: req.Profile.IgnorePatterns,
			ReplaceRules:   len(req.Profile.ReplaceRules),
			TargetPrefixes: len(req.Profile.TargetPrefixes),
		},
		Pairs: pairLimits(req.Pairs),
	})
	if violation != nil {
		writeErrorWithDetails(
			w,
			http.StatusForbidden,
			violation.Message,
			violation.Code,
			mergeMaps(
				violation.Details,
				map[string]any{"current_tier": policy.Tier},
			),
		)
		return
	}

	assertionYAML := strings.TrimSpace(req.AssertionYAML)
	if assertionYAML == "" {
		assertionYAML = defaultRuleset
	}

	var set model.AssertionSet
	if err := yaml.Unmarshal([]byte(assertionYAML), &set); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid assertion yaml: %v", err))
		return
	}
	if err := assertions.Validate(set); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	pairs := make([]model.Pair, 0, len(req.Pairs))
	for i, p := range req.Pairs {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			name = fmt.Sprintf("config-%d.cfg", i+1)
		}
		pairs = append(pairs, model.Pair{
			Name:   name,
			Before: splitConfigLines(p.Before),
			After:  splitConfigLines(p.After),
		})
	}

	result := check.Run(pairs, set)
	diffProfile := model.DiffProfile{
		OrderMode:      req.Profile.OrderMode,
		IgnorePatterns: req.Profile.IgnorePatterns,
		TargetPrefixes: req.Profile.TargetPrefixes,
	}
	for _, rule := range req.Profile.ReplaceRules {
		diffProfile.ReplaceRules = append(diffProfile.ReplaceRules, model.ReplaceRule{
			Pattern:     rule.Pattern,
			Replacement: rule.Replacement,
		})
	}
	structured, err := structureddiff.BuildFiles(pairs, diffProfile)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid diff profile: %v", err))
		return
	}
	resp := checkResponse{
		OK:             result.Summary.FilesFailed == 0,
		ExitCode:       0,
		Report:         result,
		StructuredDiff: structured,
		Stdout:         "",
		Stderr:         "",
	}
	if !resp.OK {
		resp.ExitCode = 1
	}
	if req.WantMarkdown {
		resp.Markdown = report.RenderMarkdown(result)
	}
	if req.WantReport {
		if !license.HasFeature(policy, license.FeatureReportExport) {
			writeErrorWithDetails(
				w,
				http.StatusForbidden,
				"report export is available in pro tier only",
				"LICENSE_FEATURE_BLOCKED",
				map[string]any{
					"feature":       license.FeatureReportExport,
					"required_tier": "pro",
					"current_tier":  policy.Tier,
				},
			)
			return
		}
		diffReport := report.BuildDiffReport(structured, result, diffProfile, time.Now().UTC())
		resp.DiffReport = &diffReport
	}

	writeJSON(w, http.StatusOK, resp)
}

func handleLicenseStatus(w http.ResponseWriter, opts Options) {
	status := resolveLicenseStatus(opts)
	writeJSON(w, http.StatusOK, status)
}

func handleLicenseApply(w http.ResponseWriter, r *http.Request, opts Options) {
	var req applyLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if strings.TrimSpace(req.LicenseEnvelope) == "" {
		writeError(w, http.StatusBadRequest, "licenseEnvelope is required")
		return
	}

	if _, applyErr := validateLicenseEnvelopeBeforeSave(req.LicenseEnvelope, opts, time.Now().UTC()); applyErr != nil {
		writeErrorWithDetails(w, applyErr.status, applyErr.message, applyErr.code, applyErr.details)
		return
	}

	if err := os.MkdirAll(filepath.Dir(opts.LicenseFile), 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(opts.LicenseFile), "license-save-*.tmp")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write([]byte(req.LicenseEnvelope)); err != nil {
		tmpFile.Close()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tmpFile.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := os.Rename(tmpPath, opts.LicenseFile); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type licenseApplyError struct {
	status  int
	code    string
	message string
	details map[string]any
}

func validateLicenseEnvelopeBeforeSave(
	envelope string,
	opts Options,
	now time.Time,
) (license.Status, *licenseApplyError) {
	if len([]byte(envelope)) > maxLicenseEnvelopeBytes {
		return license.Status{}, &licenseApplyError{
			status:  http.StatusBadRequest,
			code:    "LICENSE_ENVELOPE_TOO_LARGE",
			message: "licenseEnvelope exceeds size limit",
			details: map[string]any{"limit": maxLicenseEnvelopeBytes},
		}
	}
	if !fileExists(opts.PublicKeyFile) {
		return license.Status{}, &licenseApplyError{
			status:  http.StatusInternalServerError,
			code:    "LICENSE_PUBLIC_KEY_MISSING",
			message: "public key file is not configured",
		}
	}
	if err := os.MkdirAll(filepath.Dir(opts.LicenseFile), 0o755); err != nil {
		return license.Status{}, &licenseApplyError{
			status:  http.StatusInternalServerError,
			code:    "LICENSE_STORAGE_ERROR",
			message: err.Error(),
		}
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(opts.LicenseFile), "license-validate-*.json")
	if err != nil {
		return license.Status{}, &licenseApplyError{
			status:  http.StatusInternalServerError,
			code:    "LICENSE_STORAGE_ERROR",
			message: err.Error(),
		}
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write([]byte(envelope)); err != nil {
		tmpFile.Close()
		return license.Status{}, &licenseApplyError{
			status:  http.StatusInternalServerError,
			code:    "LICENSE_STORAGE_ERROR",
			message: err.Error(),
		}
	}
	if err := tmpFile.Close(); err != nil {
		return license.Status{}, &licenseApplyError{
			status:  http.StatusInternalServerError,
			code:    "LICENSE_STORAGE_ERROR",
			message: err.Error(),
		}
	}

	status, err := license.Verify(tmpPath, opts.PublicKeyFile, now)
	if err != nil || !status.Valid {
		reason := status.Error
		if strings.TrimSpace(reason) == "" && err != nil {
			reason = err.Error()
		}
		if strings.TrimSpace(reason) == "" {
			reason = "license verification failed"
		}
		return license.Status{}, &licenseApplyError{
			status:  http.StatusBadRequest,
			code:    "LICENSE_ENVELOPE_INVALID",
			message: "licenseEnvelope is not valid",
			details: map[string]any{"reason": reason},
		}
	}

	return status, nil
}

func splitConfigLines(text string) []string {
	scanner := bufio.NewScanner(strings.NewReader(text))
	lines := make([]string, 0, 256)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) == 0 {
		return []string{}
	}
	return lines
}

func spaHandler(web fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		cleanPath := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if cleanPath == "." {
			cleanPath = "index.html"
		}
		if hasFile(web, cleanPath) {
			serveEmbeddedFile(w, r, web, cleanPath)
			return
		}
		if shouldReturnNotFound(cleanPath) {
			http.NotFound(w, r)
			return
		}

		data, err := fs.ReadFile(web, "index.html")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "embedded index.html not found")
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		http.ServeContent(w, r, "index.html", time.Now(), bytes.NewReader(data))
	}
}

func serveEmbeddedFile(w http.ResponseWriter, r *http.Request, root fs.FS, cleanPath string) {
	data, err := fs.ReadFile(root, cleanPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if path.Ext(cleanPath) == ".html" || strings.HasPrefix(cleanPath, "_nuxt/") {
		w.Header().Set("Cache-Control", "no-store")
	}
	if contentType := mime.TypeByExtension(path.Ext(cleanPath)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	http.ServeContent(w, r, path.Base(cleanPath), time.Now(), bytes.NewReader(data))
}

func hasFile(root fs.FS, p string) bool {
	if strings.HasSuffix(p, "/") {
		return false
	}
	f, err := root.Open(p)
	if err != nil {
		return false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func shouldReturnNotFound(cleanPath string) bool {
	if strings.HasPrefix(cleanPath, "_nuxt/") {
		return true
	}
	return path.Ext(cleanPath) != ""
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeErrorWithDetails(w, status, message, "", nil)
}

func writeErrorWithDetails(w http.ResponseWriter, status int, message, code string, details map[string]any) {
	if status < 400 {
		status = http.StatusBadRequest
	}
	payload := map[string]any{
		"statusCode":    status,
		"statusMessage": message,
	}
	if strings.TrimSpace(code) != "" {
		payload["code"] = code
	}
	if len(details) > 0 {
		payload["details"] = details
	}
	writeJSON(w, status, payload)
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil || !errors.Is(err, os.ErrNotExist)
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *loggingResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(data)
	w.size += n
	return n, err
}

func withAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(lrw, r)
		if lrw.status == 0 {
			lrw.status = http.StatusOK
		}
		base := fmt.Sprintf("%s %s -> %d %dB (%s)", r.Method, r.URL.Path, lrw.status, lrw.size, time.Since(start))
		if lrw.status >= 400 || strings.HasPrefix(r.URL.Path, "/_nuxt/") {
			contentType := lrw.Header().Get("Content-Type")
			referer := r.Referer()
			userAgent := r.UserAgent()
			if referer == "" {
				referer = "-"
			}
			if userAgent == "" {
				userAgent = "-"
			}
			if contentType == "" {
				contentType = "-"
			}
			log.Printf("%s ct=%s referer=%q ua=%q", base, contentType, referer, userAgent)
			return
		}
		log.Print(base)
	})
}

func resolveLicenseStatus(opts Options) license.Status {
	if !fileExists(opts.LicenseFile) || !fileExists(opts.PublicKeyFile) {
		return license.ApplyPolicy(license.Status{
			Valid:     false,
			Plan:      "free",
			Error:     "license or public key not configured",
			IsExpired: false,
		})
	}

	status, err := license.Verify(opts.LicenseFile, opts.PublicKeyFile, time.Now().UTC())
	if err != nil {
		return license.ApplyPolicy(status)
	}
	return license.ApplyPolicy(status)
}

func pairLimits(pairs []pairInput) []license.CheckPair {
	limits := make([]license.CheckPair, 0, len(pairs))
	for _, p := range pairs {
		limits = append(limits, license.CheckPair{
			BeforeBytes: len([]byte(p.Before)),
			AfterBytes:  len([]byte(p.After)),
		})
	}
	return limits
}

func mergeMaps(base map[string]any, extra map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
