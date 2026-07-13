//go:build windows

package main

import (
	"fmt"
	"syscall"
)

// detachAttr: --detach relies on Unix sessions; on Windows, run flawless
// in a second terminal instead.
func detachAttr() (*syscall.SysProcAttr, error) {
	return nil, fmt.Errorf("--detach is not supported on Windows; run flawless in a second terminal")
}
