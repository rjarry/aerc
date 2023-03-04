package ipc

import (
	"bufio"
	"errors"
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
	handler  Handler
}

func StartServer(handler Handler) (*AercServer, error) {
	sockpath := path.Join(xdg.RuntimeDir(), "aerc.sock")
	// remove the socket if it is not connected to a session
	if err := ConnectAndExec(nil); err != nil {
		os.Remove(sockpath)
	}
	log.Debugf("Starting Unix server: %s", sockpath)
	l, err := net.Listen("unix", sockpath)
	if err != nil {
		return nil, err
	}
	as := &AercServer{listener: l, handler: handler}
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
			// allow up to 1 minute between commands
			err = conn.SetDeadline(time.Now().Add(1 * time.Minute))
			if err != nil {
				log.Errorf("unix:%d failed to update deadline: %v", clientId, err)
			}
			msg, err := DecodeRequest(scanner.Bytes())
			log.Tracef("unix:%d got message %s", clientId, scanner.Text())
			if err != nil {
				log.Errorf("unix:%d failed to parse request: %v", clientId, err)
				continue
			}

			response := as.handleMessage(msg)
			result, err := response.Encode()
			if err != nil {
				log.Errorf("unix:%d failed to encode result: %v", clientId, err)
				continue
			}
			_, err = conn.Write(append(result, '\n'))
			if err != nil {
				log.Errorf("unix:%d failed to send response: %v", clientId, err)
				break
			}
		}
		log.Tracef("unix:%d closed connection", clientId)
	}
}

func (as *AercServer) handleMessage(req *Request) *Response {
	if len(req.Arguments) == 0 {
		return &Response{} // send noop success message, i.e. ping
	}
	var err error
	switch {
	case strings.HasPrefix(req.Arguments[0], "mailto:"):
		mailto, err := url.Parse(req.Arguments[0])
		if err != nil {
			return &Response{Error: err.Error()}
		}
		err = as.handler.Mailto(mailto)
		if err != nil {
			return &Response{
				Error: err.Error(),
			}
		}
	case strings.HasPrefix(req.Arguments[0], "mbox:"):
		err = as.handler.Mbox(req.Arguments[0])
		if err != nil {
			return &Response{Error: err.Error()}
		}
	default:
		return &Response{Error: "command not understood"}
	}
	return &Response{}
}
