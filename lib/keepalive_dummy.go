//+build !linux

package lib

func SetTcpKeepaliveProbes(fd, count int) error {
	return nil
}

func SetTcpKeepaliveInterval(fd, interval int) error {
	return nil
}
