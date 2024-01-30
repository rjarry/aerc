package send

import (
	"fmt"
	"net/url"
	"strings"
)

func parseScheme(uri *url.URL) (protocol string, auth string, err error) {
	protocol = ""
	auth = "plain"
	if uri.Scheme != "" {
		parts := strings.Split(uri.Scheme, "+")
		switch len(parts) {
		case 1:
			protocol = parts[0]
		case 2:
			if parts[1] == "insecure" {
				protocol = uri.Scheme
			} else {
				protocol = parts[0]
				auth = parts[1]
			}
		case 3:
			protocol = parts[0] + "+" + parts[1]
			auth = parts[2]
		default:
			return "", "", fmt.Errorf("Unknown scheme %s", uri.Scheme)
		}
	}
	return protocol, auth, nil
}
