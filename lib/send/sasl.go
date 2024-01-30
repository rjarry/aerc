package send

import (
	"fmt"
	"net/url"

	"github.com/emersion/go-sasl"
	"golang.org/x/oauth2"

	"git.sr.ht/~rjarry/aerc/lib"
)

func newSaslClient(auth string, uri *url.URL) (sasl.Client, error) {
	var saslClient sasl.Client
	switch auth {
	case "":
		fallthrough
	case "none":
		saslClient = nil
	case "login":
		password, _ := uri.User.Password()
		saslClient = sasl.NewLoginClient(uri.User.Username(), password)
	case "plain":
		password, _ := uri.User.Password()
		saslClient = sasl.NewPlainClient("", uri.User.Username(), password)
	case "oauthbearer":
		q := uri.Query()
		oauth2 := &oauth2.Config{}
		if q.Get("token_endpoint") != "" {
			oauth2.ClientID = q.Get("client_id")
			oauth2.ClientSecret = q.Get("client_secret")
			oauth2.Scopes = []string{q.Get("scope")}
			oauth2.Endpoint.TokenURL = q.Get("token_endpoint")
		}
		password, _ := uri.User.Password()
		bearer := lib.OAuthBearer{
			OAuth2:  oauth2,
			Enabled: true,
		}
		if bearer.OAuth2.Endpoint.TokenURL != "" {
			token, err := bearer.ExchangeRefreshToken(password)
			if err != nil {
				return nil, err
			}
			password = token.AccessToken
		}
		saslClient = sasl.NewOAuthBearerClient(&sasl.OAuthBearerOptions{
			Username: uri.User.Username(),
			Token:    password,
		})
	case "xoauth2":
		q := uri.Query()
		oauth2 := &oauth2.Config{}
		if q.Get("token_endpoint") != "" {
			oauth2.ClientID = q.Get("client_id")
			oauth2.ClientSecret = q.Get("client_secret")
			oauth2.Scopes = []string{q.Get("scope")}
			oauth2.Endpoint.TokenURL = q.Get("token_endpoint")
		}
		password, _ := uri.User.Password()
		bearer := lib.Xoauth2{
			OAuth2:  oauth2,
			Enabled: true,
		}
		if bearer.OAuth2.Endpoint.TokenURL != "" {
			token, err := bearer.ExchangeRefreshToken(password)
			if err != nil {
				return nil, err
			}
			password = token.AccessToken
		}
		saslClient = lib.NewXoauth2Client(uri.User.Username(), password)
	default:
		return nil, fmt.Errorf("Unsupported auth mechanism %s", auth)
	}
	return saslClient, nil
}
