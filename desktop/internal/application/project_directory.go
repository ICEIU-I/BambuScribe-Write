package application

import (
	"fmt"
	"os"
	"path/filepath"
)

// The installed app must find projects without relying on its working directory.
func defaultProjectDirectory() (string, error) {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法确定项目目录：%w", err)
	}
	directory := filepath.Join(homeDirectory, "Documents", "BambuScribe", "projects")
	if err := os.MkdirAll(directory, 0755); err != nil {
		return "", fmt.Errorf("无法创建项目目录，请检查 Documents 的访问权限：%w", err)
	}
	return directory, nil
}
