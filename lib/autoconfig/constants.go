package autoconfig

type protocol uint8

const (
	// ProtocolUnknown is returned when determining the proper protocol was
	// not possible
	ProtocolUnknown protocol = 255
	// ProtocolJMAP is returned when the given server uses the JMAP protocol
	ProtocolJMAP protocol = iota
	// ProtocolIMAP is returned when the given server uses the IMAP protocol
	ProtocolIMAP
)

type encryption uint8

const (
	// EncryptionUnknown is returned when determining the proper encryption
	// was not possible
	EncryptionUnknown encryption = 255
	// EncryptionSTARTTLS is returned when the given connection uses
	// STARTTLS
	EncryptionSTARTTLS encryption = iota
	// EncryptionTLS is returned when the given connection uses a TLS
	// wrapped connection
	EncryptionTLS
	// EncryptionInsecure is returned when a connection is not encrypted
	EncryptionInsecure
)
