package imap

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/log"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// connect establishes a new tcp connection to the imap server, logs in and
// selects the default inbox. If no error is returned, the imap client will be
// in the imap.SelectedState.
func (w *IMAPWorker) connect() (*client.Client, error) {
	var (
		conn *net.TCPConn
		err  error
		c    *client.Client
	)

	conn, err = newTCPConn(w.config.addr, w.config.connection_timeout)
	if conn == nil || err != nil {
		return nil, err
	}

	if w.config.connection_timeout > 0 {
		end := time.Now().Add(w.config.connection_timeout)
		err = conn.SetDeadline(end)
		if err != nil {
			return nil, err
		}
	}

	if w.config.keepalive_period > 0 {
		err = w.setKeepaliveParameters(conn)
		if err != nil {
			return nil, err
		}
	}

	serverName, _, _ := net.SplitHostPort(w.config.addr)
	tlsConfig := &tls.Config{ServerName: serverName}

	switch w.config.scheme {
	case "imap":
		c, err = client.New(conn)
		if err != nil {
			return nil, err
		}
		if !w.config.insecure {
			if err = c.StartTLS(tlsConfig); err != nil {
				return nil, err
			}
		}
	case "imaps":
		tlsConn := tls.Client(conn, tlsConfig)
		c, err = client.New(tlsConn)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("Unknown IMAP scheme %s", w.config.scheme)
	}

	c.ErrorLog = log.ErrorLogger()

	if w.config.user != nil {
		username := w.config.user.Username()

		// TODO: 2nd parameter false if no password is set. ask for it
		// if unset.
		password, _ := w.config.user.Password()

		if w.config.oauthBearer.Enabled {
			if err := w.config.oauthBearer.Authenticate(
				username, password, c); err != nil {
				return nil, err
			}
		} else if w.config.xoauth2.Enabled {
			if err := w.config.xoauth2.Authenticate(
				username, password, c); err != nil {
				return nil, err
			}
		} else if err := c.Login(username, password); err != nil {
			return nil, err
		}
	}

	if _, err := c.Select(imap.InboxName, false); err != nil {
		return nil, err
	}

	return c, nil
}

// newTCPConn establishes a new tcp connection. Timeout will ensure that the
// function does not hang when there is no connection. If there is a timeout,
// but a valid connection is eventually returned, ensure that it is properly
// closed.
func newTCPConn(addr string, timeout time.Duration) (*net.TCPConn, error) {
	errTCPTimeout := fmt.Errorf("tcp connection timeout")

	type tcpConn struct {
		conn *net.TCPConn
		err  error
	}

	done := make(chan tcpConn)
	go func() {
		addr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			done <- tcpConn{nil, err}
			return
		}

		newConn, err := net.DialTCP("tcp", nil, addr)
		if err != nil {
			done <- tcpConn{nil, err}
			return
		}

		done <- tcpConn{newConn, nil}
	}()

	select {
	case <-time.After(timeout):
		go func() {
			if tcpResult := <-done; tcpResult.conn != nil {
				tcpResult.conn.Close()
			}
		}()
		return nil, errTCPTimeout
	case tcpResult := <-done:
		if tcpResult.conn == nil || tcpResult.err != nil {
			return nil, tcpResult.err
		}
		return tcpResult.conn, nil
	}
}

// Set additional keepalive parameters.
// Uses new interfaces introduced in Go1.11, which let us get connection's file
// descriptor, without blocking, and therefore without uncontrolled spawning of
// threads (not goroutines, actual threads).
func (w *IMAPWorker) setKeepaliveParameters(conn *net.TCPConn) error {
	err := conn.SetKeepAlive(true)
	if err != nil {
		return err
	}
	// Idle time before sending a keepalive probe
	err = conn.SetKeepAlivePeriod(w.config.keepalive_period)
	if err != nil {
		return err
	}
	rawConn, e := conn.SyscallConn()
	if e != nil {
		return e
	}
	err = rawConn.Control(func(fdPtr uintptr) {
		fd := int(fdPtr)
		// Max number of probes before failure
		err := lib.SetTcpKeepaliveProbes(fd, w.config.keepalive_probes)
		if err != nil {
			log.Errorf("cannot set tcp keepalive probes: %v", err)
		}
		// Wait time after an unsuccessful probe
		err = lib.SetTcpKeepaliveInterval(fd, w.config.keepalive_interval)
		if err != nil {
			log.Errorf("cannot set tcp keepalive interval: %v", err)
		}
	})
	return err
}
