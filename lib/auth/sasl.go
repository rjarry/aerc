package auth

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/emersion/go-sasl"
	"golang.org/x/oauth2"
)

func ParseScheme(uri *url.URL) (protocol string, mech string, err error) {
	protocol = ""
	mech = "plain"
	if uri.Scheme != "" {
		parts := strings.Split(uri.Scheme, "+")
		if len(parts) == 0 {
			return "", "", fmt.Errorf("Unknown scheme %s", uri.Scheme)
		}
		protocol = parts[0]
		parts = slices.Delete(parts, 0, 1)
		i := slices.Index(parts, "insecure")
		if i != -1 {
			protocol += "+insecure"
			parts = slices.Delete(parts, i, i+1)
		}
		if len(parts) > 0 {
			mech = strings.Join(parts, "+")
		}
	}
	return protocol, mech, nil
}

func NewSaslClient(mech string, uri *url.URL, acct string) (sasl.Client, error) {
	var saslClient sasl.Client

	user := uri.User.Username()
	password, _ := uri.User.Password()

	switch mech {
	case "", "none":
		saslClient = nil
	case "login":
		saslClient = sasl.NewLoginClient(user, password)
	case "plain":
		saslClient = sasl.NewPlainClient("", user, password)
	case "oauthbearer", "xoauth2":
		q := uri.Query()
		o := oauth2.Config{
			ClientID:     q.Get("client_id"),
			ClientSecret: q.Get("client_secret"),
			Scopes:       strings.Split(q.Get("scope"), " "),
			Endpoint: oauth2.Endpoint{
				TokenURL: q.Get("token_endpoint"),
			},
		}
		password, err := GetAccessToken(&o, acct, mech, password)
		if err != nil {
			return nil, err
		}
		if mech == "xoauth2" {
			saslClient = NewXoauth2Client(user, password)
		} else {
			saslClient = sasl.NewOAuthBearerClient(
				&sasl.OAuthBearerOptions{
					Username: uri.User.Username(),
					Token:    password,
				},
			)
		}
	default:
		return nil, fmt.Errorf("Unsupported auth mechanism %q", mech)
	}
	return saslClient, nil
}
