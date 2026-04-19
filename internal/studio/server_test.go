package studio

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleCheckGeneratesReportWithoutLicense(t *testing.T) {
	body, err := json.Marshal(checkRequest{
		Pairs: []pairInput{
			{
				Name:   "edge.cfg",
				Before: "interface Gi0/1\n description old\n ip address 10.0.0.1 255.255.255.0\n",
				After:  "interface Gi0/1\n description new\n ip address 10.0.0.2 255.255.255.0\n",
			},
		},
		WantReport: true,
		Profile: checkProfile{
			OrderMode:      "strict",
			ReplaceRules:   []replaceRule{{Pattern: `10\.0\.0\.[0-9]+`, Replacement: "<lan-ip>"}},
			TargetPrefixes: []string{"interface"},
		},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/check", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handleCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp checkResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.DiffReport == nil {
		t.Fatal("expected diff report without license")
	}
}
