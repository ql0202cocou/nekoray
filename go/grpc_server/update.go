package grpc_server

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"grpc_server/gen"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/matsuridayo/libneko/neko_common"
)

const (
	maxUpdateSize     = 200 * 1024 * 1024
	maxSignatureSize  = 1024 * 1024
	updatePartPath    = "../nekoray.zip.part"
	updatePackagePath = "../nekoray.zip"
)

// Official release builds must set this with:
// -ldflags "-X grpc_server.updatePublicKeyBase64=<base64-ed25519-public-key>"
// Empty keys fail closed so unsigned updates cannot be installed.
var updatePublicKeyBase64 string

type updateCandidate struct {
	assetName    string
	downloadURL  string
	signatureURL string
}

var (
	updateCandidateState updateCandidate
	updateCandidateMu    sync.Mutex
)

var downloadMu sync.Mutex

func (s *BaseServer) Update(ctx context.Context, in *gen.UpdateReq) (*gen.UpdateResp, error) {
	ret := &gen.UpdateResp{}

	client := neko_common.CreateProxyHttpClient(neko_common.GetCurrentInstance())

	if in.Action == gen.UpdateAction_Check { // Check update
		updateCandidateMu.Lock()
		updateCandidateState = updateCandidate{}
		updateCandidateMu.Unlock()

		ctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/repos/MatsuriDayo/nekoray/releases", nil)
		if err != nil {
			ret.Error = err.Error()
			return ret, nil
		}
		resp, err := client.Do(req)
		if err != nil {
			ret.Error = err.Error()
			return ret, nil
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			ret.Error = fmt.Sprintf("update check failed: HTTP %d", resp.StatusCode)
			return ret, nil
		}

		v := []struct {
			HtmlUrl string `json:"html_url"`
			Assets  []struct {
				Name               string `json:"name"`
				BrowserDownloadUrl string `json:"browser_download_url"`
			} `json:"assets"`
			Prerelease bool   `json:"prerelease"`
			Body       string `json:"body"`
		}{}
		err = json.NewDecoder(resp.Body).Decode(&v)
		if err != nil {
			ret.Error = err.Error()
			return ret, nil
		}

		nowVer := strings.TrimPrefix(neko_common.Version_neko, "nekoray-")

		var search string
		if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
			search = "windows64"
		} else if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
			search = "linux64"
		} else if runtime.GOOS == "darwin" {
			search = "macos-" + runtime.GOARCH
		} else {
			ret.Error = "Not official support platform"
			return ret, nil
		}

		for _, release := range v {
			if len(release.Assets) > 0 {
				assetsByName := map[string]string{}
				for _, asset := range release.Assets {
					assetsByName[asset.Name] = asset.BrowserDownloadUrl
				}
				for _, asset := range release.Assets {
					if strings.Contains(asset.Name, nowVer) {
						return ret, nil // No update
					}
					if strings.Contains(asset.Name, search) && !strings.HasSuffix(asset.Name, ".sig") {
						if release.Prerelease && !in.CheckPreRelease {
							continue
						}
						signatureURL := assetsByName[asset.Name+".sig"]
						if signatureURL == "" {
							ret.Error = "update signature asset is missing: " + asset.Name + ".sig"
							return ret, nil
						}
						updateCandidateMu.Lock()
						updateCandidateState = updateCandidate{
							assetName:    asset.Name,
							downloadURL:  asset.BrowserDownloadUrl,
							signatureURL: signatureURL,
						}
						updateCandidateMu.Unlock()
						ret.AssetsName = asset.Name
						ret.DownloadUrl = asset.BrowserDownloadUrl
						ret.ReleaseUrl = release.HtmlUrl
						ret.ReleaseNote = release.Body
						ret.IsPreRelease = release.Prerelease
						return ret, nil // update
					}
				}
			}
		}
	} else { // Download update
		downloadMu.Lock()
		defer downloadMu.Unlock()

		updateCandidateMu.Lock()
		candidate := updateCandidateState
		updateCandidateMu.Unlock()
		if candidate.downloadURL == "" || candidate.signatureURL == "" {
			ret.Error = "?"
			return ret, nil
		}
		if _, err := updatePublicKey(); err != nil {
			ret.Error = err.Error()
			return ret, nil
		}

		os.Remove(updatePartPath)
		if err := downloadToFile(ctx, client, candidate.downloadURL, updatePartPath, maxUpdateSize); err != nil {
			ret.Error = err.Error()
			return ret, nil
		}
		signature, err := downloadBytes(ctx, client, candidate.signatureURL, maxSignatureSize)
		if err != nil {
			os.Remove(updatePartPath)
			ret.Error = err.Error()
			return ret, nil
		}
		if err := verifyUpdateSignature(updatePartPath, signature); err != nil {
			os.Remove(updatePartPath)
			ret.Error = err.Error()
			return ret, nil
		}
		os.Remove(updatePackagePath)
		if err := os.Rename(updatePartPath, updatePackagePath); err != nil {
			os.Remove(updatePartPath)
			ret.Error = err.Error()
			return ret, nil
		}
	}

	return ret, nil
}

func downloadToFile(ctx context.Context, client *http.Client, url, path string, limit int64) error {
	resp, err := doDownloadRequest(ctx, client, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.ContentLength > limit {
		return fmt.Errorf("download exceeds max size: %d bytes", resp.ContentLength)
	}

	f, err := os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	lr := &io.LimitedReader{R: resp.Body, N: limit + 1}
	n, err := io.Copy(f, lr)
	if err != nil {
		os.Remove(path)
		return err
	}
	if n > limit {
		os.Remove(path)
		return fmt.Errorf("download exceeds max size: %d bytes", limit)
	}
	if err := f.Sync(); err != nil {
		os.Remove(path)
		return err
	}
	return nil
}

func downloadBytes(ctx context.Context, client *http.Client, url string, limit int64) ([]byte, error) {
	resp, err := doDownloadRequest(ctx, client, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.ContentLength > limit {
		return nil, fmt.Errorf("download exceeds max size: %d bytes", resp.ContentLength)
	}

	lr := &io.LimitedReader{R: resp.Body, N: limit + 1}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("download exceeds max size: %d bytes", limit)
	}
	return data, nil
}

func doDownloadRequest(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	return resp, nil
}

func verifyUpdateSignature(archivePath string, signature []byte) error {
	key, err := updatePublicKey()
	if err != nil {
		return err
	}

	sig, err := normalizeUpdateSignature(signature)
	if err != nil {
		return err
	}
	archive, err := os.ReadFile(archivePath)
	if err != nil {
		return err
	}
	if !ed25519.Verify(key, archive, sig) {
		return errors.New("update signature verification failed")
	}
	return nil
}

func updatePublicKey() (ed25519.PublicKey, error) {
	keyText := strings.TrimSpace(updatePublicKeyBase64)
	if keyText == "" {
		return nil, errors.New("update signature public key is not configured")
	}
	key, err := base64.StdEncoding.DecodeString(keyText)
	if err != nil {
		return nil, fmt.Errorf("invalid update public key: %w", err)
	}
	if len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid update public key length: %d", len(key))
	}
	return ed25519.PublicKey(key), nil
}

func normalizeUpdateSignature(signature []byte) ([]byte, error) {
	sig := bytes.TrimSpace(signature)
	if len(sig) == ed25519.SignatureSize {
		return sig, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(string(sig))
	if err == nil && len(decoded) == ed25519.SignatureSize {
		return decoded, nil
	}
	return nil, fmt.Errorf("invalid update signature length: %d", len(sig))
}
