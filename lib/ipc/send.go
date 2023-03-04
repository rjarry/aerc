package ipc

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"path"

	"github.com/kyoh86/xdg"
)

func ConnectAndExec(msg string) error {
	sockpath := path.Join(xdg.RuntimeDir(), "aerc.sock")
	conn, err := net.Dial("unix", sockpath)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write([]byte(msg + "\n"))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return errors.New("No response from server")
	}
	result := scanner.Text()
	fmt.Println(result)
	return nil
}
