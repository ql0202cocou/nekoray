package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestSignAndVerifyAsset(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	asset := filepath.Join(t.TempDir(), "asset.zip")
	if err := os.WriteFile(asset, []byte("update payload"), 0644); err != nil {
		t.Fatal(err)
	}

	privateKeyB64 := base64.StdEncoding.EncodeToString(privateKey)
	publicKeyB64 := base64.StdEncoding.EncodeToString(publicKey)

	if err := signAsset(asset, privateKeyB64); err != nil {
		t.Fatal(err)
	}
	if err := verifyAsset(asset, asset+".sig", publicKeyB64); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(asset, []byte("tampered payload"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := verifyAsset(asset, asset+".sig", publicKeyB64); err == nil {
		t.Fatal("expected tampered asset verification to fail")
	}
}

func TestDecodeKeyValidation(t *testing.T) {
	if _, err := decodePublicKey(""); err == nil {
		t.Fatal("expected missing public key to fail")
	}
	if _, err := decodePrivateKey(base64.StdEncoding.EncodeToString([]byte("short"))); err == nil {
		t.Fatal("expected short private key to fail")
	}
}
