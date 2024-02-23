package ipc

type Handler interface {
	Command(args []string) error
}
