package lib

// TODO: Remove this pending merge into github.com/emersion/go-message

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/martinlindhe/base36"
)

// Generates an RFC 2822-complaint Message-Id based on the informational draft
// "Recommendations for generating Message IDs", for lack of a better
// authoritative source.
func GenerateMessageId() string {
	var (
		now   bytes.Buffer
		nonce bytes.Buffer
	)
	binary.Write(&now, binary.BigEndian, time.Now().UnixNano())
	binary.Write(&nonce, binary.BigEndian, rand.Uint64())
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}
	return fmt.Sprintf("<%s.%s@%s>",
		base36.EncodeBytes(now.Bytes()),
		base36.EncodeBytes(nonce.Bytes()),
		hostname)
}
