package autoconfig

// Config contains the discovered settings for the mailserver
type Config struct {
	Found protocol
	JMAP  Credentials
	IMAP  Credentials
	SMTP  Credentials
}

// Credentials contains the discovered settings for a protocol.
type Credentials struct {
	Encryption encryption
	Address    string
	Port       int
	Username   string
}
