package send

import (
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/emersion/go-message/mail"
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

func GetMessageIdHostname(sendWithHostname bool, from *mail.Address) (string, error) {
	if sendWithHostname {
		return os.Hostname()
	}
	if from == nil {
		// no from address present, generate a random hostname
		return strings.ToUpper(strconv.FormatInt(rand.Int63(), 36)), nil
	}
	_, domain, found := strings.Cut(from.Address, "@")
	if !found {
		return "", fmt.Errorf("Invalid address %q", from)
	}
	return domain, nil
}
