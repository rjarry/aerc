package lib

import (
	"context"
	"fmt"

	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-sasl"
	"golang.org/x/oauth2"
)

type OAuthBearer struct {
	OAuth2  *oauth2.Config
	Enabled bool
}

func (c *OAuthBearer) ExchangeRefreshToken(refreshToken string) (*oauth2.Token, error) {
	token := new(oauth2.Token)
	token.RefreshToken = refreshToken
	token.TokenType = "Bearer"
	return c.OAuth2.TokenSource(context.TODO(), token).Token()
}

func (c *OAuthBearer) Authenticate(username string, password string, client *client.Client) error {
	if ok, err := client.SupportAuth(sasl.OAuthBearer); err != nil || !ok {
		return fmt.Errorf("OAuthBearer not supported %v", err)
	}

	if c.OAuth2.Endpoint.TokenURL != "" {
		token, err := c.ExchangeRefreshToken(password)
		if err != nil {
			return err
		}
		password = token.AccessToken
	}

	saslClient := sasl.NewOAuthBearerClient(&sasl.OAuthBearerOptions{
		Username: username,
		Token:    password,
	})

	return client.Authenticate(saslClient)
}
