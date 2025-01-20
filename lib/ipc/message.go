package ipc

import "encoding/json"

// Request contains all parameters needed for the main instance to respond to
// a request.
type Request struct {
	// Arguments contains the commandline arguments. The detection of what
	// action to take is left to the receiver.
	Arguments []string `json:"arguments"`
}

// Response is used to report the results of a command.
type Response struct {
	// Error contains the success-state of the command. Error is an empty
	// string if everything ran successfully.
	Error string `json:"error"`
}

// Encode transforms the message in an easier to transfer format
func (msg *Request) Encode() ([]byte, error) {
	return json.Marshal(msg)
}

// DecodeMessage consumes a raw message and returns the message contained
// within.
func DecodeMessage(data []byte) (*Request, error) {
	msg := new(Request)
	err := json.Unmarshal(data, msg)
	return msg, err
}

// Encode transforms the message in an easier to transfer format
func (msg *Response) Encode() ([]byte, error) {
	return json.Marshal(msg)
}

// DecodeRequest consumes a raw message and returns the message contained
// within.
func DecodeRequest(data []byte) (*Request, error) {
	msg := new(Request)
	err := json.Unmarshal(data, msg)
	return msg, err
}

// DecodeResponse consumes a raw message and returns the message contained
// within.
func DecodeResponse(data []byte) (*Response, error) {
	msg := new(Response)
	err := json.Unmarshal(data, msg)
	return msg, err
}
