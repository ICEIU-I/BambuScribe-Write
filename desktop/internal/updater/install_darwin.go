//go:build darwin

package updater

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const helperFlag = "--bambuscribe-apply-update"

func installedBundle() (string, error) {
	if runtime.GOARCH != "arm64" {
		return "", ErrUnsupported
	}
	executable, err := os.Executable()
	if err != nil {
		return "", ErrLocation
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return "", ErrLocation
	}
	target := filepath.Dir(filepath.Dir(filepath.Dir(executable)))
	if filepath.Base(target) != BundleName || executable != filepath.Join(target, "Contents", "MacOS", ExecutableName) || strings.HasPrefix(target, "/Volumes/") || strings.Contains(target, "/AppTranslocation/") {
		return "", ErrLocation
	}
	if stat, err := os.Lstat(target); err != nil || !stat.IsDir() {
		return "", ErrLocation
	}
	return target, nil
}

func plist(ctx context.Context, bundle, key string) (string, error) {
	data, err := exec.CommandContext(ctx, "/usr/libexec/PlistBuddy", "-c", "Print :"+key, filepath.Join(bundle, "Contents", "Info.plist")).Output()
	return strings.TrimSpace(string(data)), err
}

func verifyBundle(ctx context.Context, bundle, target, version string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for key, want := range map[string]string{"CFBundleShortVersionString": version, "CFBundleExecutable": ExecutableName} {
		got, err := plist(ctx, bundle, key)
		if err != nil || got != want {
			return ErrPackage
		}
	}
	wantID, err := plist(ctx, target, "CFBundleIdentifier")
	if err != nil || wantID == "" {
		return ErrPackage
	}
	gotID, err := plist(ctx, bundle, "CFBundleIdentifier")
	if err != nil || gotID != wantID {
		return ErrPackage
	}
	if err := exec.CommandContext(ctx, "/usr/bin/codesign", "--verify", "--deep", "--strict", bundle).Run(); err != nil {
		return ErrPackage
	}
	binary := filepath.Join(bundle, "Contents", "MacOS", ExecutableName)
	if err := exec.CommandContext(ctx, "/usr/bin/lipo", "-verify_arch", "arm64", binary).Run(); err != nil {
		return ErrPackage
	}
	actual, err := exec.CommandContext(ctx, binary, "--version").Output()
	if err != nil || strings.TrimSpace(string(actual)) != version {
		return ErrPackage
	}
	return nil
}

func (c *Client) Prepare(ctx context.Context, release Release, progress func(Progress)) (*Prepared, error) {
	target, err := installedBundle()
	if err != nil {
		return nil, err
	}
	return c.prepareAt(ctx, release, target, progress)
}

func (c *Client) prepareAt(ctx context.Context, release Release, target string, progress func(Progress)) (*Prepared, error) {
	dir, err := os.MkdirTemp(filepath.Dir(target), ".bambuscribe-update-")
	if err != nil {
		return nil, ErrLocation
	}
	p := &Prepared{Target: target, WorkDir: dir}
	ok := false
	defer func() {
		if !ok {
			p.Cleanup()
		}
	}()
	archive, err := c.Download(ctx, release, dir, progress)
	if err != nil {
		return nil, err
	}
	if progress != nil {
		progress(Progress{Phase: "verifying", Percent: 100})
	}
	version := strings.TrimPrefix(release.Tag, "v")
	p.Bundle, err = extractBundle(archive, dir, version)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err = verifyBundle(ctx, p.Bundle, target, version); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ok = true
	return p, nil
}

func (p *Prepared) Start(ctx context.Context, version string) error {
	cmd := exec.Command(filepath.Join(p.Bundle, "Contents", "MacOS", ExecutableName), helperFlag, strconv.Itoa(os.Getpid()), p.Target, p.WorkDir, version)
	logfile, err := os.OpenFile(filepath.Join(p.WorkDir, "install.log"), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return ErrInstall
	}
	cmd.Stdout, cmd.Stderr = logfile, logfile
	err = cmd.Start()
	logfile.Close()
	if err != nil {
		return ErrInstall
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if _, err := os.Stat(filepath.Join(p.WorkDir, "ready")); err == nil {
				return nil
			}
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			<-exited
			return ctx.Err()
		case <-timer.C:
			_ = cmd.Process.Kill()
			<-exited
			return ErrInstall
		case <-exited:
			return ErrInstall
		}
	}
}

// HandleHelper runs before Wails starts, in the verified replacement binary.
// It never touches the user's configuration or projects.
func HandleHelper(args []string) (bool, error) {
	if len(args) == 0 || args[0] != helperFlag {
		return false, nil
	}
	if len(args) != 5 {
		return true, ErrInstall
	}
	pid, err := strconv.Atoi(args[1])
	target, dir, version := args[2], args[3], args[4]
	if err != nil || pid <= 1 || pid == os.Getpid() || !filepath.IsAbs(target) || !filepath.IsAbs(dir) || filepath.Base(target) != BundleName || filepath.Dir(dir) != filepath.Dir(target) || !strings.HasPrefix(filepath.Base(dir), ".bambuscribe-update-") {
		return true, ErrInstall
	}
	p := &Prepared{Target: target, Bundle: filepath.Join(dir, BundleName), WorkDir: dir}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := verifyBundle(ctx, p.Bundle, target, version); err != nil {
		return true, err
	}
	if err := os.WriteFile(filepath.Join(dir, "ready"), []byte("ready\n"), 0600); err != nil {
		return true, ErrInstall
	}
	for syscall.Kill(pid, 0) != syscall.ESRCH {
		select {
		case <-ctx.Done():
			return true, ErrInstall
		case <-time.After(100 * time.Millisecond):
		}
	}
	err = p.replace(func(target string) error {
		args := []string{"-n", target}
		if dir := os.Getenv("BAMBUSCRIBE_CONFIG_DIR"); dir != "" {
			args = append(args, "--env", "BAMBUSCRIBE_CONFIG_DIR="+dir)
		}
		return exec.Command("/usr/bin/open", args...).Run()
	})
	if err != nil {
		// Preserve the backup and install.log if a rename could not be rolled back.
		_ = exec.Command("/usr/bin/osascript", "-e", `display alert "更新未完成" message "原版本已保留。请重新打开应用，或从 GitHub Releases 下载后安装。"`).Run()
		return true, fmt.Errorf("%w: %v", ErrInstall, err)
	}
	return true, nil
}
