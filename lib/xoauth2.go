//
// This code is derived from the go-sasl library.
//
// Copyright (c) 2016 emersion
// Copyright (c) 2022, Oracle and/or its affiliates.
//
// SPDX-License-Identifier: MIT

package lib

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-sasl"
	"golang.org/x/oauth2"
)

// An XOAUTH2 error.
type Xoauth2Error struct {
	Status  string `json:"status"`
	Schemes string `json:"schemes"`
	Scope   string `json:"scope"`
}

// Implements error.
func (err *Xoauth2Error) Error() string {
	return fmt.Sprintf("XOAUTH2 authentication error (%v)", err.Status)
}

type xoauth2Client struct {
	Username string
	Token    string
}

func (a *xoauth2Client) Start() (mech string, ir []byte, err error) {
	mech = "XOAUTH2"
	ir = []byte("user=" + a.Username + "\x01auth=Bearer " + a.Token + "\x01\x01")
	return
}

func (a *xoauth2Client) Next(challenge []byte) ([]byte, error) {
	// Server sent an error response
	xoauth2Err := &Xoauth2Error{}
	if err := json.Unmarshal(challenge, xoauth2Err); err != nil {
		return nil, err
	} else {
		return nil, xoauth2Err
	}
}

// An implementation of the XOAUTH2 authentication mechanism, as
// described in https://developers.google.com/gmail/xoauth2_protocol.
func NewXoauth2Client(username, token string) sasl.Client {
	return &xoauth2Client{username, token}
}

type Xoauth2 struct {
	OAuth2  *oauth2.Config
	Enabled bool
}

func (c *Xoauth2) ExchangeRefreshToken(refreshToken string) (*oauth2.Token, error) {
	token := new(oauth2.Token)
	token.RefreshToken = refreshToken
	token.TokenType = "Bearer"
	return c.OAuth2.TokenSource(context.TODO(), token).Token()
}

func (c *Xoauth2) Authenticate(username string, password string, client *client.Client) error {
	if ok, err := client.SupportAuth("XOAUTH2"); err != nil || !ok {
		return fmt.Errorf("Xoauth2 not supported %w", err)
	}

	if c.OAuth2.Endpoint.TokenURL != "" {
		token, err := c.ExchangeRefreshToken(password)
		if err != nil {
			return err
		}
		password = token.AccessToken
	}

	saslClient := NewXoauth2Client(username, password)

	return client.Authenticate(saslClient)
}
