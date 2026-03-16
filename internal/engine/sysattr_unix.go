//go:build linux || darwin

package engine

import "syscall"

func sysAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
