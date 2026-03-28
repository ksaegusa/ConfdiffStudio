package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

type Envelope struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

type Claims struct {
	LicenseID    string   `json:"license_id"`
	Plan         string   `json:"plan"`
	IssuedAt     string   `json:"issued_at"`
	ExpiresAt    string   `json:"expires_at"`
	Features     []string `json:"features"`
	SeatLimit    int      `json:"seat_limit,omitempty"`
	CustomerName string   `json:"customer_name,omitempty"`
}

type Status struct {
	Valid        bool     `json:"valid"`
	Plan         string   `json:"plan"`
	Tier         string   `json:"tier"`
	Expiry       string   `json:"expiry"`
	Features     []string `json:"features"`
	Limits       Limits   `json:"limits"`
	Entitlements []string `json:"entitlements"`
	LicenseID    string   `json:"license_id,omitempty"`
	Error        string   `json:"error,omitempty"`
	Customer     string   `json:"customer_name,omitempty"`
	SeatLimit    int      `json:"seat_limit,omitempty"`
	IssuedAt     string   `json:"issued_at,omitempty"`
	IsExpired    bool     `json:"is_expired"`
}

func Verify(licensePath, publicKeyPath string, now time.Time) (Status, error) {
	envData, err := os.ReadFile(licensePath)
	if err != nil {
		return fallbackStatus(fmt.Sprintf("cannot read license file: %v", err)), err
	}
	pubData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fallbackStatus(fmt.Sprintf("cannot read public key file: %v", err)), err
	}

	var env Envelope
	if err := json.Unmarshal(envData, &env); err != nil {
		return fallbackStatus(fmt.Sprintf("invalid license envelope: %v", err)), err
	}

	payload, err := base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		return fallbackStatus(fmt.Sprintf("invalid payload encoding: %v", err)), err
	}
	sig, err := base64.StdEncoding.DecodeString(env.Signature)
	if err != nil {
		return fallbackStatus(fmt.Sprintf("invalid signature encoding: %v", err)), err
	}

	pubRaw, err := base64.StdEncoding.DecodeString(string(bytesTrim(pubData)))
	if err != nil {
		return fallbackStatus(fmt.Sprintf("invalid public key encoding: %v", err)), err
	}
	if len(pubRaw) != ed25519.PublicKeySize {
		return fallbackStatus("invalid public key size"), fmt.Errorf("invalid public key size")
	}

	if !ed25519.Verify(ed25519.PublicKey(pubRaw), payload, sig) {
		return fallbackStatus("signature verification failed"), fmt.Errorf("signature verification failed")
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return fallbackStatus(fmt.Sprintf("invalid claims payload: %v", err)), err
	}

	expiryTime, err := time.Parse(time.RFC3339, claims.ExpiresAt)
	if err != nil {
		return fallbackStatus(fmt.Sprintf("invalid expiry timestamp: %v", err)), err
	}

	isExpired := now.After(expiryTime)
	status := Status{
		Valid:     !isExpired,
		Plan:      claims.Plan,
		Expiry:    claims.ExpiresAt,
		Features:  claims.Features,
		LicenseID: claims.LicenseID,
		Customer:  claims.CustomerName,
		SeatLimit: claims.SeatLimit,
		IssuedAt:  claims.IssuedAt,
		IsExpired: isExpired,
	}

	if status.Plan == "" {
		status.Plan = "free"
	}
	if isExpired {
		status.Error = "license expired"
	}
	status = ApplyPolicy(status)
	return status, nil
}

func fallbackStatus(message string) Status {
	return ApplyPolicy(Status{
		Valid:     false,
		Plan:      "free",
		Error:     message,
		IsExpired: false,
	})
}

func ApplyPolicy(status Status) Status {
	policy := PolicyForStatus(status)
	status.Plan = strings.ToLower(strings.TrimSpace(status.Plan))
	if status.Plan == "" {
		status.Plan = "free"
	}
	status.Tier = policy.Tier
	status.Limits = policy.Limits
	status.Entitlements = make([]string, 0, len(policy.Entitlements))

	// If signed claims contain feature list, enforce intersection with known capabilities.
	if status.Valid && len(status.Features) > 0 {
		filtered := make([]string, 0, len(status.Features))
		for _, feature := range status.Features {
			if HasFeature(policy, feature) {
				filtered = append(filtered, feature)
				status.Entitlements = append(status.Entitlements, feature)
			}
		}
		sort.Strings(status.Entitlements)
		status.Features = filtered
		return status
	}

	for feature := range policy.Entitlements {
		status.Entitlements = append(status.Entitlements, feature)
	}
	sort.Strings(status.Entitlements)
	status.Features = append([]string{}, status.Entitlements...)
	return status
}

func bytesTrim(b []byte) []byte {
	start := 0
	end := len(b)
	for start < end && (b[start] == ' ' || b[start] == '\n' || b[start] == '\r' || b[start] == '\t') {
		start++
	}
	for end > start && (b[end-1] == ' ' || b[end-1] == '\n' || b[end-1] == '\r' || b[end-1] == '\t') {
		end--
	}
	return b[start:end]
}
