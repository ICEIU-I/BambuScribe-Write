package updater

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Progress struct {
	Phase   string `json:"phase"`
	Percent int    `json:"percent"`
}

func (c *Client) Download(ctx context.Context, r Release, dir string, progress func(Progress)) (string, error) {
	if err := r.validate(); err != nil {
		return "", err
	}
	name := packageName(strings.TrimPrefix(r.Tag, "v"))
	checksums, _ := r.asset("SHA256SUMS")
	resp, err := c.get(ctx, checksums.URL)
	if err != nil {
		return "", err
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 65537))
	resp.Body.Close()
	if err != nil || len(data) > 65536 {
		return "", ErrPackage
	}
	want, err := checksumFor(string(data), name)
	if err != nil {
		return "", err
	}
	asset, _ := r.asset(name)
	resp, err = c.get(ctx, asset.URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	filename := filepath.Join(dir, "update.zip")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInstall, err)
	}
	ok := false
	defer func() {
		f.Close()
		if !ok {
			os.Remove(filename)
		}
	}()
	hash := sha256.New()
	var downloaded int64
	last := -1
	buffer := make([]byte, 64<<10)
	for {
		n, readErr := resp.Body.Read(buffer)
		if n > 0 {
			downloaded += int64(n)
			if downloaded > asset.Size {
				return "", ErrPackage
			}
			if _, err := io.MultiWriter(f, hash).Write(buffer[:n]); err != nil {
				return "", ErrInstall
			}
			percent := int(downloaded * 100 / asset.Size)
			if percent != last && progress != nil {
				progress(Progress{Phase: "downloading", Percent: percent})
				last = percent
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			return "", ErrNetwork
		}
	}
	if downloaded != asset.Size {
		return "", ErrPackage
	}
	if hex.EncodeToString(hash.Sum(nil)) != want {
		return "", ErrChecksum
	}
	if err := f.Close(); err != nil {
		return "", ErrInstall
	}
	ok = true
	return filename, nil
}

func checksumFor(data, name string) (string, error) {
	found := ""
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.TrimPrefix(strings.TrimPrefix(fields[1], "*"), "./") != name {
			continue
		}
		if found != "" {
			return "", ErrPackage
		}
		value, err := hex.DecodeString(fields[0])
		if err != nil || len(value) != sha256.Size {
			return "", ErrPackage
		}
		found = strings.ToLower(fields[0])
	}
	if found == "" {
		return "", ErrPackage
	}
	return found, nil
}

const BundleName = "BambuScribe Write.app"
const ExecutableName = "BambuScribe Write"

// Extract only the application, refusing links, traversal and oversized ZIPs.
func extractBundle(archive, dir, version string) (string, error) {
	z, err := zip.OpenReader(archive)
	if err != nil {
		return "", ErrPackage
	}
	defer z.Close()
	prefix := strings.TrimSuffix(packageName(version), ".zip") + "/" + BundleName + "/"
	var total uint64
	if len(z.File) > 10000 {
		return "", ErrPackage
	}
	for _, f := range z.File {
		if path.IsAbs(f.Name) || strings.Contains(f.Name, "\\") || path.Clean(f.Name) != strings.TrimSuffix(f.Name, "/") {
			return "", ErrPackage
		}
		if f.Mode()&(os.ModeSymlink|os.ModeDevice|os.ModeNamedPipe|os.ModeSocket) != 0 {
			return "", ErrPackage
		}
		if f.UncompressedSize64 > (512<<20)-total {
			return "", ErrPackage
		}
		total += f.UncompressedSize64
		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}
		rel := strings.TrimPrefix(f.Name, prefix)
		if rel == "" {
			continue
		}
		dest := filepath.Join(dir, BundleName, filepath.FromSlash(rel))
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dest, 0755); err != nil {
				return "", ErrInstall
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return "", ErrInstall
		}
		input, err := f.Open()
		if err != nil {
			return "", ErrPackage
		}
		mode := os.FileMode(0644)
		if f.Mode().Perm()&0111 != 0 {
			mode = 0755
		}
		output, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			input.Close()
			return "", ErrInstall
		}
		_, copyErr := io.Copy(output, io.LimitReader(input, int64(f.UncompressedSize64)+1))
		closeErr := output.Close()
		input.Close()
		if copyErr != nil || closeErr != nil {
			return "", ErrPackage
		}
	}
	bundle := filepath.Join(dir, BundleName)
	if _, err := os.Stat(filepath.Join(bundle, "Contents", "MacOS", ExecutableName)); err != nil {
		return "", ErrPackage
	}
	return bundle, nil
}
