package lib

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/kyoh86/xdg"
)

type AercServer struct {
	logger   *log.Logger
	listener net.Listener
	OnMailto func(addr *url.URL) error
}

func StartServer(logger *log.Logger) (*AercServer, error) {
	sockpath := path.Join(xdg.RuntimeDir(), "aerc.sock")
	l, err := net.Listen("unix", sockpath)
	if err != nil {
		return nil, err
	}
	as := &AercServer{
		logger:   logger,
		listener: l,
	}
	// TODO: stash clients and close them on exit... bleh racey
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				// TODO: Something more useful, in some cases, on wednesdays,
				// after 2 PM, I guess?
				as.logger.Printf("Closing Unix server: %v", err)
				return
			}
			go as.handleClient(conn)
		}
	}()
	return as, nil
}

func (as *AercServer) Close() {
	as.listener.Close()
}

var lastId int64 = 0 // access via atomic

func (as *AercServer) handleClient(conn net.Conn) {
	clientId := atomic.AddInt64(&lastId, 1)
	as.logger.Printf("Accepted Unix connection %d", clientId)
	scanner := bufio.NewScanner(conn)
	conn.SetDeadline(time.Now().Add(1 * time.Minute))
	for scanner.Scan() {
		conn.SetDeadline(time.Now().Add(1 * time.Minute))
		msg := scanner.Text()
		as.logger.Printf("unix:%d: got message %s", clientId, msg)
		if !strings.ContainsRune(msg, ':') {
			conn.Write([]byte("error: invalid command\n"))
			continue
		}
		prefix := msg[:strings.IndexRune(msg, ':')]
		switch prefix {
		case "mailto":
			mailto, err := url.Parse(msg)
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("error: %v\n", err)))
				break
			}
			if as.OnMailto != nil {
				err = as.OnMailto(mailto)
			}
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("result: %v\n", err)))
			} else {
				conn.Write([]byte(fmt.Sprint("result: success\n")))
			}
		}
	}
	as.logger.Printf("Closed Unix connection %d", clientId)
}

func ConnectAndExec(msg string) error {
	sockpath := path.Join(xdg.RuntimeDir(), "aerc.sock")
	conn, err := net.Dial("unix", sockpath)
	if err != nil {
		return err
	}
	conn.Write([]byte(msg + "\n"))
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return errors.New("No response from server")
	}
	result := scanner.Text()
	fmt.Println(result)
	return nil
}
