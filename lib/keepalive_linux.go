//+build linux

package lib

import (
	"syscall"
)

func SetTcpKeepaliveProbes(fd, count int) error {
	return syscall.SetsockoptInt(
		fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPCNT, count)
}

func SetTcpKeepaliveInterval(fd, interval int) error {
	return syscall.SetsockoptInt(
		fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPINTVL, interval)
}
