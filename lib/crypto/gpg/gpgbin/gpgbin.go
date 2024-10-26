package gpgbin

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/pinentry"
	"git.sr.ht/~rjarry/aerc/models"
)

// gpg represents a gpg command with buffers attached to stdout and stderr
type gpg struct {
	cmd    *exec.Cmd
	stdout bytes.Buffer
	stderr bytes.Buffer
}

// newGpg creates a new gpg command with buffers attached
func newGpg(stdin io.Reader, args []string) *gpg {
	g := new(gpg)
	g.cmd = exec.Command("gpg", "--status-fd", "2", "--log-file", "/dev/null", "--batch")
	g.cmd.Args = append(g.cmd.Args, args...)
	g.cmd.Stdin = stdin
	g.cmd.Stdout = &g.stdout
	g.cmd.Stderr = &g.stderr

	pinentry.SetCmdEnv(g.cmd)

	return g
}

// fields returns the field name from --status-fd output. See:
// https://github.com/gpg/gnupg/blob/master/doc/DETAILS
func field(s string) string {
	tokens := strings.SplitN(s, " ", 3)
	if tokens[0] == "[GNUPG:]" {
		return tokens[1]
	}
	return ""
}

// getIdentity returns the identity of the given key
func getIdentity(key uint64) string {
	fpr := fmt.Sprintf("%X", key)
	cmd := exec.Command("gpg", "--with-colons", "--batch", "--list-keys", fpr)

	var outbuf strings.Builder
	cmd.Stdout = &outbuf
	err := cmd.Run()
	if err != nil {
		log.Errorf("gpg: failed to get identity: %v", err)
		return ""
	}
	out := strings.Split(outbuf.String(), "\n")
	for _, line := range out {
		if strings.HasPrefix(line, "uid") {
			flds := strings.Split(line, ":")
			return flds[9]
		}
	}
	return ""
}

// getKeyId returns the 16 digit key id, if key exists
func getKeyId(s string, private bool) string {
	cmd := exec.Command("gpg", "--with-colons", "--batch")
	listArg := "--list-keys"
	if private {
		listArg = "--list-secret-keys"
	}
	cmd.Args = append(cmd.Args, listArg, s)

	var outbuf strings.Builder
	cmd.Stdout = &outbuf
	err := cmd.Run()
	if err != nil {
		log.Errorf("gpg: failed to get key ID: %v", err)
		return ""
	}
	out := strings.Split(outbuf.String(), "\n")
	for _, line := range out {
		if strings.HasPrefix(line, "fpr") {
			flds := strings.Split(line, ":")
			id := flds[9]
			return id[len(id)-16:]
		}
	}
	return ""
}

// longKeyToUint64 returns a uint64 version of the given key
func longKeyToUint64(key string) (uint64, error) {
	fpr := string(key[len(key)-16:])
	fprUint64, err := strconv.ParseUint(fpr, 16, 64)
	if err != nil {
		return 0, err
	}
	return fprUint64, nil
}

// parse parses the output of gpg --status-fd
func parseStatusFd(r io.Reader, md *models.MessageDetails) error {
	var err error
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if field(line) == "PLAINTEXT_LENGTH" {
			continue
		}
		log.Tracef(line)

		switch field(line) {
		case "ENC_TO":
			md.IsEncrypted = true
		case "DECRYPTION_KEY":
			md.DecryptedWithKeyId, err = parseDecryptionKey(line)
			md.DecryptedWith = getIdentity(md.DecryptedWithKeyId)
			if err != nil {
				return err
			}
		case "DECRYPTION_FAILED":
			return EncryptionFailed
		case "NEWSIG":
			md.IsSigned = true
		case "GOODSIG":
			t := strings.SplitN(line, " ", 4)
			md.SignedByKeyId, err = longKeyToUint64(t[2])
			if err != nil {
				return err
			}
			md.SignedBy = t[3]
		case "BADSIG":
			t := strings.SplitN(line, " ", 4)
			md.SignedByKeyId, err = longKeyToUint64(t[2])
			if err != nil {
				return err
			}
			md.SignatureError = "gpg: invalid signature"
			md.SignedBy = t[3]
		case "EXPSIG":
			t := strings.SplitN(line, " ", 4)
			md.SignedByKeyId, err = longKeyToUint64(t[2])
			if err != nil {
				return err
			}
			md.SignatureError = "gpg: expired signature"
			md.SignedBy = t[3]
		case "EXPKEYSIG":
			t := strings.SplitN(line, " ", 4)
			md.SignedByKeyId, err = longKeyToUint64(t[2])
			if err != nil {
				return err
			}
			md.SignatureError = "gpg: signature made with expired key"
			md.SignedBy = t[3]
		case "REVKEYSIG":
			t := strings.SplitN(line, " ", 4)
			md.SignedByKeyId, err = longKeyToUint64(t[2])
			if err != nil {
				return err
			}
			md.SignatureError = "gpg: signature made with revoked key"
			md.SignedBy = t[3]
		case "ERRSIG":
			t := strings.SplitN(line, " ", 9)
			md.SignedByKeyId, err = longKeyToUint64(t[2])
			if err != nil {
				return err
			}
			if t[7] == "9" {
				md.SignatureError = "gpg: missing public key"
			}
			if t[7] == "4" {
				md.SignatureError = "gpg: unsupported algorithm"
			}
			md.SignedBy = "(unknown signer)"
		case "INV_RECP":
			t := strings.SplitN(line, " ", 4)
			if t[2] == "10" {
				return fmt.Errorf("gpg: public key of %s is not trusted", t[3])
			}
		case "SIG_CREATED":
			fields := strings.Split(line, " ")
			micalg, err := strconv.Atoi(fields[4])
			if err != nil {
				return MicalgNotFound
			}
			md.Micalg = micalgs[micalg]
		case "VALIDSIG":
			fields := strings.Split(line, " ")
			micalg, err := strconv.Atoi(fields[9])
			if err != nil {
				return MicalgNotFound
			}
			md.Micalg = micalgs[micalg]
		case "NODATA":
			t := strings.SplitN(line, " ", 3)
			if t[2] == "4" {
				md.SignatureError = "gpg: no signature packet found"
			}
			if t[2] == "1" {
				return NoValidOpenPgpData
			}
		case "FAILURE":
			return fmt.Errorf("%s", strings.TrimPrefix(line, "[GNUPG:] "))
		}
	}
	return nil
}

// parseDecryptionKey returns primary key from DECRYPTION_KEY line
func parseDecryptionKey(l string) (uint64, error) {
	key := strings.Split(l, " ")[3]
	fpr := string(key[len(key)-16:])
	fprUint64, err := longKeyToUint64(fpr)
	if err != nil {
		return 0, err
	}
	getIdentity(fprUint64)
	return fprUint64, nil
}

type StatusFdParsingError int32

const (
	EncryptionFailed StatusFdParsingError = iota + 1
	MicalgNotFound
	NoValidOpenPgpData
)

func (err StatusFdParsingError) Error() string {
	switch err {
	case EncryptionFailed:
		return "gpg: decryption failed"
	case MicalgNotFound:
		return "gpg: micalg not found"
	case NoValidOpenPgpData:
		return "gpg: no valid OpenPGP data found"
	default:
		return "gpg: unknown status fd parsing error"
	}
}

// micalgs represent hash algorithms for signatures. These are ignored by many
// email clients, but can be used as an additional verification so are sent.
// Both gpgmail and pgpmail implementations in aerc check for matching micalgs
var micalgs = map[int]string{
	1:  "pgp-md5",
	2:  "pgp-sha1",
	3:  "pgp-ripemd160",
	8:  "pgp-sha256",
	9:  "pgp-sha384",
	10: "pgp-sha512",
	11: "pgp-sha224",
}
