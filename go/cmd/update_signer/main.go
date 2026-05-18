package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	genKey := flag.Bool("gen-key", false, "generate an Ed25519 key pair")
	verify := flag.Bool("verify", false, "verify signature instead of signing")
	privateKeyB64 := flag.String("private-key", os.Getenv("NKR_UPDATE_PRIVATE_KEY_BASE64"), "base64 Ed25519 private key")
	publicKeyB64 := flag.String("public-key", os.Getenv("NKR_UPDATE_PUBLIC_KEY_BASE64"), "base64 Ed25519 public key")
	signaturePath := flag.String("sig", "", "signature path for verify mode")
	flag.Parse()

	if *genKey {
		return generateKeyPair()
	}

	args := flag.Args()
	if len(args) == 0 {
		return errors.New("usage: update_signer [-gen-key] [-verify -public-key <base64> -sig <file.sig>] <asset>...")
	}

	if *verify {
		if len(args) != 1 {
			return errors.New("verify mode accepts exactly one asset")
		}
		if *signaturePath == "" {
			return errors.New("verify mode requires -sig")
		}
		return verifyAsset(args[0], *signaturePath, *publicKeyB64)
	}

	for _, asset := range args {
		if err := signAsset(asset, *privateKeyB64); err != nil {
			return err
		}
	}
	return nil
}

func generateKeyPair() error {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	fmt.Println("NKR_UPDATE_PUBLIC_KEY_BASE64=" + base64.StdEncoding.EncodeToString(publicKey))
	fmt.Println("NKR_UPDATE_PRIVATE_KEY_BASE64=" + base64.StdEncoding.EncodeToString(privateKey))
	return nil
}

func signAsset(assetPath, privateKeyB64 string) error {
	privateKey, err := decodePrivateKey(privateKeyB64)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(assetPath)
	if err != nil {
		return err
	}
	signature := ed25519.Sign(privateKey, data)
	sigPath := assetPath + ".sig"
	if err := os.WriteFile(sigPath, []byte(base64.StdEncoding.EncodeToString(signature)+"\n"), 0644); err != nil {
		return err
	}
	fmt.Println("signed", assetPath, "->", sigPath)
	return nil
}

func verifyAsset(assetPath, sigPath, publicKeyB64 string) error {
	publicKey, err := decodePublicKey(publicKeyB64)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(assetPath)
	if err != nil {
		return err
	}
	signature, err := os.ReadFile(sigPath)
	if err != nil {
		return err
	}
	sig, err := normalizeSignature(signature)
	if err != nil {
		return err
	}
	if !ed25519.Verify(publicKey, data, sig) {
		return errors.New("signature verification failed")
	}
	fmt.Println("verified", assetPath)
	return nil
}

func decodePrivateKey(keyB64 string) (ed25519.PrivateKey, error) {
	key, err := decodeBase64(strings.TrimSpace(keyB64), "private key")
	if err != nil {
		return nil, err
	}
	if len(key) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key length: %d", len(key))
	}
	return ed25519.PrivateKey(key), nil
}

func decodePublicKey(keyB64 string) (ed25519.PublicKey, error) {
	key, err := decodeBase64(strings.TrimSpace(keyB64), "public key")
	if err != nil {
		return nil, err
	}
	if len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length: %d", len(key))
	}
	return ed25519.PublicKey(key), nil
}

func normalizeSignature(signature []byte) ([]byte, error) {
	sig := []byte(strings.TrimSpace(string(signature)))
	if len(sig) == ed25519.SignatureSize {
		return sig, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(string(sig))
	if err == nil && len(decoded) == ed25519.SignatureSize {
		return decoded, nil
	}
	return nil, fmt.Errorf("invalid signature length: %d", len(sig))
}

func decodeBase64(value, name string) ([]byte, error) {
	if value == "" {
		return nil, fmt.Errorf("missing %s", name)
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", name, err)
	}
	return decoded, nil
}
