package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestVerifyValidLicense(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	dir := t.TempDir()
	pubPath := filepath.Join(dir, "pub.key")
	licPath := filepath.Join(dir, "license.json")

	claims := Claims{
		LicenseID: "lic-1",
		Plan:      "pro",
		IssuedAt:  time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		ExpiresAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		Features:  []string{FeatureReportExport},
	}

	payload, _ := json.Marshal(claims)
	sig := ed25519.Sign(priv, payload)
	env := Envelope{Payload: base64.StdEncoding.EncodeToString(payload), Signature: base64.StdEncoding.EncodeToString(sig)}
	encodedEnv, _ := json.Marshal(env)

	if err := os.WriteFile(pubPath, []byte(base64.StdEncoding.EncodeToString(pub)), 0o644); err != nil {
		t.Fatalf("write pub key: %v", err)
	}
	if err := os.WriteFile(licPath, encodedEnv, 0o644); err != nil {
		t.Fatalf("write license: %v", err)
	}

	status, err := Verify(licPath, pubPath, time.Now().UTC())
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !status.Valid || status.Plan != "pro" {
		t.Fatalf("unexpected status: %+v", status)
	}
	if len(status.Features) == 0 || status.Features[0] != FeatureReportExport {
		t.Fatalf("unexpected features: %+v", status.Features)
	}
}
