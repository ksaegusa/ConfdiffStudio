package studio

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ksaegusa/ConfdiffStudio/internal/license"
)

func TestValidateLicenseEnvelopeBeforeSaveRejectsInvalidEnvelope(t *testing.T) {
	opts, validEnvelope := makeLicenseFixture(t, time.Now().Add(time.Hour))
	if err := os.WriteFile(opts.LicenseFile, []byte(validEnvelope), 0o644); err != nil {
		t.Fatalf("write existing license: %v", err)
	}

	before, err := os.ReadFile(opts.LicenseFile)
	if err != nil {
		t.Fatalf("read existing license: %v", err)
	}

	_, applyErr := validateLicenseEnvelopeBeforeSave(`{"payload":"broken"}`, opts, time.Now().UTC())
	if applyErr == nil {
		t.Fatal("expected invalid envelope error")
	}
	if applyErr.code != "LICENSE_ENVELOPE_INVALID" {
		t.Fatalf("unexpected error code: %+v", applyErr)
	}

	after, err := os.ReadFile(opts.LicenseFile)
	if err != nil {
		t.Fatalf("read existing license after failure: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("existing license should remain unchanged")
	}
}

func TestHandleLicenseApplyRejectsExpiredLicense(t *testing.T) {
	opts, _ := makeLicenseFixture(t, time.Now().Add(time.Hour))
	expiredEnvelope := signEnvelope(t, filepath.Join(filepath.Dir(opts.PublicKeyFile), "private.key"), time.Now().Add(-time.Hour))

	body, err := json.Marshal(applyLicenseRequest{LicenseEnvelope: expiredEnvelope})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/license/apply", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handleLicenseApply(rec, req, opts)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(opts.LicenseFile); err == nil {
		t.Fatal("expired license must not be saved")
	}
}

func TestHandleLicenseApplySavesValidLicense(t *testing.T) {
	opts, validEnvelope := makeLicenseFixture(t, time.Now().Add(time.Hour))

	body, err := json.Marshal(applyLicenseRequest{LicenseEnvelope: validEnvelope})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/license/apply", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handleLicenseApply(rec, req, opts)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(opts.LicenseFile); err != nil {
		t.Fatalf("expected saved license: %v", err)
	}
	status, err := license.Verify(opts.LicenseFile, opts.PublicKeyFile, time.Now().UTC())
	if err != nil || !status.Valid {
		t.Fatalf("saved license should verify: status=%+v err=%v", status, err)
	}
}

func makeLicenseFixture(t *testing.T, expiry time.Time) (Options, string) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	dir := t.TempDir()
	publicKeyPath := filepath.Join(dir, "public.key")
	privateKeyPath := filepath.Join(dir, "private.key")
	licensePath := filepath.Join(dir, "license.json")

	if err := os.WriteFile(publicKeyPath, []byte(base64.StdEncoding.EncodeToString(pub)), 0o644); err != nil {
		t.Fatalf("write public key: %v", err)
	}
	if err := os.WriteFile(privateKeyPath, []byte(base64.StdEncoding.EncodeToString(priv)), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}

	return Options{
		LicenseFile:   licensePath,
		PublicKeyFile: publicKeyPath,
	}, signEnvelope(t, privateKeyPath, expiry)
}

func signEnvelope(t *testing.T, privateKeyPath string, expiry time.Time) string {
	t.Helper()

	privRawText, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatalf("read private key: %v", err)
	}
	privRaw, err := base64.StdEncoding.DecodeString(string(privRawText))
	if err != nil {
		t.Fatalf("decode private key: %v", err)
	}

	claims := license.Claims{
		LicenseID: "test-license",
		Plan:      "pro",
		IssuedAt:  time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		ExpiresAt: expiry.UTC().Format(time.RFC3339),
		Features:  []string{license.FeatureReportExport},
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}

	sig := ed25519.Sign(ed25519.PrivateKey(privRaw), payload)
	env, err := json.Marshal(license.Envelope{
		Payload:   base64.StdEncoding.EncodeToString(payload),
		Signature: base64.StdEncoding.EncodeToString(sig),
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	return string(env)
}
