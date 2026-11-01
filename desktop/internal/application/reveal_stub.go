//go:build !darwin

package application

import "fmt"

func reveal(string) error {
	return fmt.Errorf("当前系统不支持打开所在目录")
}
