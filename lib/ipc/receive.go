package ipc

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"git.sr.ht/~rjarry/aerc/log"
	"github.com/kyoh86/xdg"
)

type AercServer struct {
	listener net.Listener

	OnMailto func(addr *url.URL) error
	OnMbox   func(source string) error
}

func StartServer() (*AercServer, error) {
	sockpath := path.Join(xdg.RuntimeDir(), "aerc.sock")
	// remove the socket if it is not connected to a session
	if err := ConnectAndExec(""); err != nil {
		os.Remove(sockpath)
	}
	log.Debugf("Starting Unix server: %s", sockpath)
	l, err := net.Listen("unix", sockpath)
	if err != nil {
		return nil, err
	}
	as := &AercServer{listener: l}
	go as.Serve()

	return as, nil
}

func (as *AercServer) Close() {
	as.listener.Close()
}

var lastId int64 = 0 // access via atomic

func (as *AercServer) Serve() {
	defer log.PanicHandler()

	for {
		conn, err := as.listener.Accept()
		switch {
		case errors.Is(err, net.ErrClosed):
			log.Infof("shutting down UNIX listener")
			return
		case err != nil:
			log.Errorf("ipc: accepting connection failed: %v", err)
			continue
		}

		defer conn.Close()
		clientId := atomic.AddInt64(&lastId, 1)
		log.Debugf("unix:%d accepted connection", clientId)
		scanner := bufio.NewScanner(conn)
		err = conn.SetDeadline(time.Now().Add(1 * time.Minute))
		if err != nil {
			log.Errorf("unix:%d failed to set deadline: %v", clientId, err)
		}
		for scanner.Scan() {
			err = conn.SetDeadline(time.Now().Add(1 * time.Minute))
			if err != nil {
				log.Errorf("unix:%d failed to update deadline: %v", clientId, err)
			}
			msg := scanner.Text()
			log.Tracef("unix:%d got message %s", clientId, msg)

			_, err = conn.Write([]byte(as.handleMessage(msg)))
			if err != nil {
				log.Errorf("unix:%d failed to send response: %v", clientId, err)
				break
			}
		}
		log.Tracef("unix:%d closed connection", clientId)
	}
}

func (as *AercServer) handleMessage(msg string) string {
	if !strings.ContainsRune(msg, ':') {
		return "error: invalid command\n"
	}
	prefix := msg[:strings.IndexRune(msg, ':')]
	var err error
	switch prefix {
	case "mailto":
		mailto, err := url.Parse(msg)
		if err != nil {
			return fmt.Sprintf("error: %v\n", err)
		}
		if as.OnMailto != nil {
			err = as.OnMailto(mailto)
			if err != nil {
				return fmt.Sprintf("mailto failed: %v\n", err)
			}
		}
	case "mbox":
		if as.OnMbox != nil {
			err = as.OnMbox(msg)
			if err != nil {
				return fmt.Sprintf("mbox failed: %v\n", err)
			}
		}
	}
	return "result: success\n"
}
