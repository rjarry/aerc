package ipc

import (
	"bufio"
	"errors"
	"fmt"
	"net"

	"git.sr.ht/~rjarry/aerc/lib/xdg"
)

func ConnectAndExec(args []string) error {
	sockpath := xdg.RuntimePath("aerc.sock")
	conn, err := net.Dial("unix", sockpath)
	if err != nil {
		return err
	}
	defer conn.Close()

	req, err := (&Request{Arguments: args}).Encode()
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	_, err = conn.Write(append(req, '\n'))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return errors.New("No response from server")
	}
	resp, err := DecodeResponse(scanner.Bytes())
	if err != nil {
		return err
	}

	// TODO: handle this in a more elegant manner
	if resp.Error != "" {
		fmt.Println("result: ", resp.Error)
	}

	return nil
}
