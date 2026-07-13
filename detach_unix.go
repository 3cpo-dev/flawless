//go:build !windows

package main

import "syscall"

// detachAttr returns the process attributes that put a --detach child in
// its own session, so it survives the terminal closing.
func detachAttr() (*syscall.SysProcAttr, error) {
	return &syscall.SysProcAttr{Setsid: true}, nil
}
