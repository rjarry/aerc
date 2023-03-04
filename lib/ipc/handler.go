package ipc

import "net/url"

type Handler interface {
	Mailto(addr *url.URL) error
	Mbox(source string) error
	Command(args []string) error
}
