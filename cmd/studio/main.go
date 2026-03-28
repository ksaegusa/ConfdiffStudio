package main

import (
	"embed"
	"flag"
	"log"
	"net/http"

	"github.com/ksaegusa/ConfdiffStudio/internal/studio"
)

//go:embed all:web/dist
var webAssets embed.FS

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address")
	licenseFile := flag.String("license-file", "./.local/license/license.json", "license file path")
	publicKeyFile := flag.String("public-key-file", "./.local/license/public.key", "public key file path")
	flag.Parse()

	handler, err := studio.NewHandler(webAssets, studio.Options{
		LicenseFile:   *licenseFile,
		PublicKeyFile: *publicKeyFile,
	})
	if err != nil {
		log.Fatalf("init handler: %v", err)
	}

	server := &http.Server{
		Addr:    *addr,
		Handler: handler,
	}
	log.Printf("Confdiff Studio listening on http://%s", *addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
