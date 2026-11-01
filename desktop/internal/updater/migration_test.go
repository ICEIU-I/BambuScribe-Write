package updater

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestVersionOneUsesConsolidatedRepository(t *testing.T) {
	const root = "https://github.com/ICEIU-I/BambuScribe-Write/releases/download/v1.0.0/"
	release := Release{Tag: "v1.0.0", Assets: []Asset{
		{Name: "BambuScribe-Write-1.0.0-macos-arm64.zip", URL: root + "BambuScribe-Write-1.0.0-macos-arm64.zip", Size: 100},
		{Name: "SHA256SUMS", URL: root + "SHA256SUMS", Size: 100},
	}}
	c := NewClient()
	c.HTTP = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://api.github.com/repos/ICEIU-I/BambuScribe-Write/releases?per_page=30" {
			t.Fatalf("wrong update repository: %s", r.URL)
		}
		body, _ := json.Marshal([]Release{release})
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
	})}
	info, next, err := c.Check(context.Background(), "0.1.8")
	if err != nil || next == nil || !info.Available || info.Version != "1.0.0" {
		t.Fatalf("major upgrade: %+v, %v", info, err)
	}
	info, next, err = c.Check(context.Background(), "1.0.0")
	if err != nil || next != nil || info.Available {
		t.Fatalf("same version must not reinstall: %+v, %v", info, err)
	}
	// The old repository is an upgrade bridge, not a trusted source for future releases.
	release.Assets[0].URL = strings.Replace(release.Assets[0].URL, "BambuScribe-Write/releases", "BambuScribe-Write-Releases/releases", 1)
	if release.validate() == nil {
		t.Fatal("accepted an asset from the retired repository")
	}
}
