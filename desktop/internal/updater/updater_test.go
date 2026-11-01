package updater

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func testRelease(version string, data []byte) Release {
	root := "https://github.com/" + Repository + "/releases/download/v" + version + "/"
	return Release{Tag: "v" + version, Assets: []Asset{{Name: packageName(version), URL: root + packageName(version), Size: int64(len(data))}, {Name: "SHA256SUMS", URL: root + "SHA256SUMS", Size: 100}}}
}

func downloadClient(data []byte, checksum string) *Client {
	return &Client{HTTP: &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		body := data
		if strings.HasSuffix(r.URL.Path, "/SHA256SUMS") {
			body = []byte(checksum)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body))}, nil
	})}}
}

func TestCheckOrdersNumericVersionsAndRejectsUntrustedAssets(t *testing.T) {
	r := []Release{testRelease("0.2.0", []byte("a")), testRelease("0.10.0", []byte("a")), testRelease("1.0.0", []byte("a"))}
	r[2].Draft = true
	r[1].Prerelease = true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _ = json.NewEncoder(w).Encode(r) }))
	defer server.Close()
	c := &Client{HTTP: server.Client(), Endpoint: server.URL}
	info, release, err := c.Check(context.Background(), "0.1.1")
	if err != nil || release == nil || info.Version != "0.10.0" || !info.Available {
		t.Fatalf("%+v %v", info, err)
	}
	info, release, err = c.Check(context.Background(), "0.10.0")
	if err != nil || release != nil || info.Available {
		t.Fatalf("same-version update: %+v %v", info, err)
	}
	r[1].Assets[0].URL = "https://example.com/malicious.zip"
	if _, _, err := c.Check(context.Background(), "0.1.1"); !errors.Is(err, ErrPackage) {
		t.Fatalf("accepted a foreign download: %v", err)
	}
}

func TestVersionsNeverDowngradeOrAcceptInvalidTags(t *testing.T) {
	for _, v := range []string{"0.1.0", "0.1.1", "v0.1.1-beta", "1.2", "1.2.3.4", "01.2.3", "1.2.-3", "1.2.+3", "9999999999999999999999.0.0"} {
		if newer(v, "0.1.1") {
			t.Errorf("accepted %q", v)
		}
	}
	if !newer("v0.1.10", "0.1.9") {
		t.Fatal("numeric version ordering failed")
	}
}

func TestChecksumMismatchAndTruncationNeverLeaveAnInstaller(t *testing.T) {
	data := []byte("verified package")
	hash := sha256.Sum256(data)
	sums := hex.EncodeToString(hash[:]) + "  " + packageName("0.1.2") + "\n"
	for _, tc := range []struct {
		name     string
		body     []byte
		sums     string
		expected error
	}{
		{"success", data, sums, nil},
		{"mismatch", []byte("tampered package"), sums, ErrChecksum},
		{"truncated", data[:2], sums, ErrPackage},
		{"missing checksum", data, "", ErrPackage},
		{"duplicate checksum", data, sums + sums, ErrPackage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			r := testRelease("0.1.2", data)
			if tc.name == "mismatch" {
				r.Assets[0].Size = int64(len(tc.body))
			}
			last := 0
			file, err := downloadClient(tc.body, tc.sums).Download(context.Background(), r, dir, func(p Progress) {
				if p.Percent < last || p.Percent > 100 {
					t.Error("invalid progress")
				}
				last = p.Percent
			})
			if !errors.Is(err, tc.expected) {
				t.Fatalf("%v, want %v", err, tc.expected)
			}
			if tc.expected != nil {
				if _, err := os.Stat(filepath.Join(dir, "update.zip")); !os.IsNotExist(err) {
					t.Fatal("failed installer remains")
				}
			} else if got, err := os.ReadFile(file); err != nil || !bytes.Equal(got, data) || last != 100 {
				t.Fatal("successful download changed")
			}
		})
	}
}

func TestHTTPFailureAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(403) }))
	defer server.Close()
	c := &Client{HTTP: server.Client(), Endpoint: server.URL}
	if _, _, err := c.Check(context.Background(), "0.1.1"); !errors.Is(err, ErrNetwork) {
		t.Fatalf("%v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := c.Check(ctx, "0.1.1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
}

func makeArchive(t *testing.T, name string, mode os.FileMode) string {
	t.Helper()
	var data bytes.Buffer
	z := zip.NewWriter(&data)
	h := &zip.FileHeader{Name: name, Method: zip.Deflate}
	h.SetMode(mode)
	w, err := z.CreateHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("executable"))
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "archive.zip")
	if err := os.WriteFile(file, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestExtractionRejectsTraversalAndLinksAndPreservesExecutable(t *testing.T) {
	root := strings.TrimSuffix(packageName("0.1.2"), ".zip") + "/" + BundleName + "/"
	for _, tc := range []struct {
		name  string
		mode  os.FileMode
		valid bool
	}{
		{root + "Contents/MacOS/" + ExecutableName, 0755, true},
		{root + "../../escaped", 0644, false},
		{"/tmp/escaped", 0644, false},
		{root + "Contents/MacOS/" + ExecutableName, os.ModeSymlink | 0755, false},
		{root + "Contents\\escaped", 0644, false},
	} {
		t.Run(fmt.Sprintf("%s-%d", tc.name, tc.mode), func(t *testing.T) {
			bundle, err := extractBundle(makeArchive(t, tc.name, tc.mode), t.TempDir(), "0.1.2")
			if !tc.valid {
				if !errors.Is(err, ErrPackage) {
					t.Fatalf("unsafe archive accepted: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			stat, err := os.Stat(filepath.Join(bundle, "Contents", "MacOS", ExecutableName))
			if err != nil || stat.Mode().Perm()&0111 == 0 {
				t.Fatal("lost executable mode")
			}
		})
	}
}

func TestReplacementRestoresOldAppWhenMoveOrLaunchFails(t *testing.T) {
	for _, failure := range []string{"none", "move", "launch"} {
		t.Run(failure, func(t *testing.T) {
			dir := t.TempDir()
			p := &Prepared{Target: filepath.Join(dir, BundleName), WorkDir: filepath.Join(dir, ".bambuscribe-update-test")}
			p.Bundle = filepath.Join(p.WorkDir, BundleName)
			for _, name := range []string{p.Target, p.Bundle} {
				if err := os.MkdirAll(name, 0755); err != nil {
					t.Fatal(err)
				}
			}
			_ = os.WriteFile(filepath.Join(p.Target, "version"), []byte("old"), 0644)
			_ = os.WriteFile(filepath.Join(p.Bundle, "version"), []byte("new"), 0644)
			if failure == "move" {
				_ = os.RemoveAll(p.Bundle)
			}
			launched := 0
			err := p.replace(func(path string) error {
				launched++
				if failure == "launch" && launched == 1 {
					return errors.New("launch failure")
				}
				return nil
			})
			want := "new"
			if failure != "none" {
				want = "old"
				if !errors.Is(err, ErrInstall) {
					t.Fatalf("%v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(p.Target, "version"))
			if err != nil || string(got) != want {
				t.Fatalf("old bundle lost: %s %v", got, err)
			}
			if failure == "launch" && launched != 2 {
				t.Fatal("old bundle was not relaunched")
			}
		})
	}
}
