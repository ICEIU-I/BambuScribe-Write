// Package updater downloads and verifies published application bundles.
package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const Repository = "ICEIU-I/BambuScribe-Write-Releases"
const ReleasesURL = "https://github.com/" + Repository + "/releases"
const maxPackageSize = 256 << 20

var (
	ErrNetwork     = errors.New("update service unavailable")
	ErrPackage     = errors.New("invalid update package")
	ErrChecksum    = errors.New("update checksum mismatch")
	ErrLocation    = errors.New("install the app in a writable Applications directory")
	ErrUnsupported = errors.New("updates require macOS Apple Silicon")
	ErrInstall     = errors.New("update installation failed")
	ErrBusy        = errors.New("another update is running")
)

type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

type Release struct {
	Tag        string  `json:"tag_name"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Body       string  `json:"body"`
	Assets     []Asset `json:"assets"`
}

type Info struct {
	Current   string `json:"current"`
	Version   string `json:"version"`
	Available bool   `json:"available"`
	Notes     string `json:"notes"`
	URL       string `json:"url"`
}

type Client struct {
	HTTP     *http.Client
	Endpoint string
}

func NewClient() *Client {
	return &Client{HTTP: &http.Client{Timeout: 10 * time.Minute}, Endpoint: "https://api.github.com/repos/" + Repository + "/releases?per_page=30"}
}

func versionParts(v string) ([3]uint64, bool) {
	var result [3]uint64
	p := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(p) != 3 {
		return result, false
	}
	for i, s := range p {
		if s == "" || (len(s) > 1 && s[0] == '0') {
			return result, false
		}
		for _, c := range s {
			if c < '0' || c > '9' {
				return result, false
			}
		}
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return result, false
		}
		result[i] = n
	}
	return result, true
}

func newer(a, b string) bool {
	x, okX := versionParts(a)
	y, okY := versionParts(b)
	if !okX || !okY {
		return false
	}
	for i := range x {
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	return false
}

func packageName(version string) string { return "BambuScribe-Write-" + version + "-macos-arm64.zip" }

func (r Release) asset(name string) (Asset, bool) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a, true
		}
	}
	return Asset{}, false
}

func (r Release) validate() error {
	v := strings.TrimPrefix(r.Tag, "v")
	if _, valid := versionParts(v); !valid || r.Draft {
		return ErrPackage
	}
	for _, name := range []string{packageName(v), "SHA256SUMS"} {
		a, ok := r.asset(name)
		u, err := url.Parse(a.URL)
		prefix := "/" + Repository + "/releases/download/" + r.Tag + "/"
		if !ok || err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != prefix+name || a.Size <= 0 || a.Size > maxPackageSize {
			return ErrPackage
		}
	}
	return nil
}

func (c *Client) get(ctx context.Context, address string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetwork, err)
	}
	req.Header.Set("User-Agent", "BambuScribe-Write-Updater")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%w: %v", ErrNetwork, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%w: HTTP %d", ErrNetwork, resp.StatusCode)
	}
	return resp, nil
}

func (c *Client) Check(ctx context.Context, current string) (Info, *Release, error) {
	info := Info{Current: current, Version: current, URL: ReleasesURL}
	resp, err := c.get(ctx, c.Endpoint)
	if err != nil {
		return info, nil, err
	}
	defer resp.Body.Close()
	var releases []Release
	if err = json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&releases); err != nil {
		return info, nil, ErrNetwork
	}
	var latest *Release
	// The first published version is a test release. Follow numeric versions
	// from this repository, including releases marked as a macOS test build.
	for i := range releases {
		r := &releases[i]
		if r.Draft || !newer(r.Tag, info.Version) {
			continue
		}
		info.Version = strings.TrimPrefix(r.Tag, "v")
		latest = r
	}
	if latest == nil {
		return info, nil, nil
	}
	if err := latest.validate(); err != nil {
		return info, nil, err
	}
	info.Available, info.Notes = true, latest.Body
	info.URL = ReleasesURL + "/tag/" + latest.Tag
	return info, latest, nil
}
