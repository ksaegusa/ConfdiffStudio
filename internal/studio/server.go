package studio

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/ksaegusa/ConfdiffStudio/internal/assertions"
	"github.com/ksaegusa/ConfdiffStudio/internal/check"
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

func NewHandler(staticFS embed.FS) (http.Handler, error) {
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
		handleCheck(w, r)
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

func handleCheck(w http.ResponseWriter, r *http.Request) {
	var req checkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if len(req.Pairs) == 0 {
		writeError(w, http.StatusBadRequest, "pairs are required")
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
		diffReport := report.BuildDiffReport(structured, result, diffProfile, time.Now().UTC())
		resp.DiffReport = &diffReport
	}

	writeJSON(w, http.StatusOK, resp)
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
