//go:build darwin

package updater

import (
	"bambuscribe/desktop/internal/buildinfo"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Run against a packaged, signed bundle without changing the installed app.
func TestPackagedMacOSUpdate(t *testing.T) {
	archive := os.Getenv("BAMBUSCRIBE_UPDATE_TEST_ZIP")
	if archive == "" {
		t.Skip("set BAMBUSCRIBE_UPDATE_TEST_ZIP to verify a packaged update")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	version := buildinfo.Version
	dir := t.TempDir()
	target, err := extractBundle(archive, filepath.Join(dir, "installed"), version)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	sums := hex.EncodeToString(hash[:]) + "  " + packageName(version) + "\n"
	client := downloadClient(data, sums)
	p, err := client.prepareAt(ctx, testRelease(version, data), target, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Cleanup()
	if err := p.replace(func(bundle string) error {
		got, err := exec.CommandContext(ctx, filepath.Join(bundle, "Contents", "MacOS", ExecutableName), "--version").Output()
		if err == nil && strings.TrimSpace(string(got)) != version {
			t.Fatal("wrong updated executable")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p.WorkDir); !os.IsNotExist(err) {
		t.Fatal("staging directory remains after replacement")
	}
}

func TestHelperRejectsUnexpectedArguments(t *testing.T) {
	if handled, err := HandleHelper([]string{"--version"}); handled || err != nil {
		t.Fatal("normal launch entered helper")
	}
	for _, args := range [][]string{
		{helperFlag},
		{helperFlag, "0", "/tmp/" + BundleName, "/tmp/.bambuscribe-update-test", "0.1.1"},
		{helperFlag, "123", "/tmp/other.app", "/tmp/.bambuscribe-update-test", "0.1.1"},
		{helperFlag, "123", "/tmp/" + BundleName, "/elsewhere/.bambuscribe-update-test", "0.1.1"},
	} {
		if handled, err := HandleHelper(args); !handled || err == nil {
			t.Fatal("invalid helper arguments accepted")
		}
	}
}
