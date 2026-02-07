package auth

import (
	"context"
	"fmt"
	"os"
	"path"

	"git.sr.ht/~rjarry/aerc/lib/xdg"
	"golang.org/x/oauth2"
)

func exchangeRefreshToken(o *oauth2.Config, refreshToken string) (*oauth2.Token, error) {
	token := new(oauth2.Token)
	token.RefreshToken = refreshToken
	token.TokenType = "Bearer"
	return o.TokenSource(context.TODO(), token).Token()
}

func tokenCachePath(account, mech string) string {
	return xdg.CachePath("aerc", account+"-"+mech+".token")
}

func saveRefreshToken(refreshToken, account, mech string) error {
	p := tokenCachePath(account, mech)
	if err := os.MkdirAll(path.Dir(p), 0o700); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(refreshToken), 0o600)
}

func getRefreshToken(account, mech string) (string, error) {
	buf, err := os.ReadFile(tokenCachePath(account, mech))
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func GetAccessToken(
	o *oauth2.Config, account, mech, password string,
) (string, error) {
	if o.Endpoint.TokenURL == "" {
		return password, nil
	}
	usedCache := false
	if r, err := getRefreshToken(account, mech); err == nil && len(r) > 0 {
		password = string(r)
		usedCache = true
	}

	token, err := exchangeRefreshToken(o, password)
	if err != nil {
		if usedCache {
			return "", fmt.Errorf("%w: try deleting %s",
				err, tokenCachePath(account, mech))
		}
		return "", err
	}
	if err := saveRefreshToken(token.RefreshToken, account, mech); err != nil {
		return "", err
	}
	return token.AccessToken, nil
}
