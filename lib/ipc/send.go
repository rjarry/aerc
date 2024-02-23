package ipc

import (
	"bufio"
	"errors"
	"fmt"
	"net"

	"git.sr.ht/~rjarry/aerc/lib/xdg"
)

func ConnectAndExec(args []string) (*Response, error) {
	sockpath := xdg.RuntimePath("aerc.sock")
	conn, err := net.Dial("unix", sockpath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	req, err := (&Request{Arguments: args}).Encode()
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	_, err = conn.Write(append(req, '\n'))
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return nil, errors.New("No response from server")
	}
	resp, err := DecodeResponse(scanner.Bytes())
	if err != nil {
		return nil, err
	}

	return resp, nil
}
