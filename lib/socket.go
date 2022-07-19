package lib

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

	"git.sr.ht/~rjarry/aerc/logging"
	"github.com/kyoh86/xdg"
)

type AercServer struct {
	listener net.Listener
	OnMailto func(addr *url.URL) error
	OnMbox   func(source string) error
}

func StartServer() (*AercServer, error) {
	sockpath := path.Join(xdg.RuntimeDir(), "aerc.sock")
	// remove the socket if it already exists
	os.Remove(sockpath)
	logging.Infof("Starting Unix server: %s", sockpath)
	l, err := net.Listen("unix", sockpath)
	if err != nil {
		return nil, err
	}
	as := &AercServer{listener: l}
	// TODO: stash clients and close them on exit... bleh racey
	go func() {
		defer logging.PanicHandler()

		for {
			conn, err := l.Accept()
			if err != nil {
				// TODO: Something more useful, in some cases, on wednesdays,
				// after 2 PM, I guess?
				logging.Errorf("Closing Unix server: %v", err)
				return
			}
			go func() {
				defer logging.PanicHandler()

				as.handleClient(conn)
			}()
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
	logging.Debugf("unix:%d accepted connection", clientId)
	scanner := bufio.NewScanner(conn)
	conn.SetDeadline(time.Now().Add(1 * time.Minute))
	for scanner.Scan() {
		conn.SetDeadline(time.Now().Add(1 * time.Minute))
		msg := scanner.Text()
		logging.Debugf("unix:%d got message %s", clientId, msg)
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
				conn.Write([]byte("result: success\n"))
			}
		case "mbox":
			var err error
			if as.OnMbox != nil {
				err = as.OnMbox(msg)
			}
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("result: %v\n", err)))
			} else {
				conn.Write([]byte("result: success\n"))
			}
		}
	}
	logging.Debugf("unix:%d closed connection", clientId)
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
