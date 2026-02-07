package send

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/emersion/go-message/mail"
)

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
