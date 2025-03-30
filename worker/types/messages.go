package types

import (
	"context"
	"io"
	"time"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-message/mail"
)

type WorkerMessage interface {
	InResponseTo() WorkerMessage
	getId() int64
	setId(id int64)
	Account() string
	setAccount(string)
}

type Message struct {
	inResponseTo WorkerMessage
	id           int64
	acct         string
}

func RespondTo(msg WorkerMessage) Message {
	return Message{
		inResponseTo: msg,
	}
}

func (m Message) InResponseTo() WorkerMessage {
	return m.inResponseTo
}

func (m Message) getId() int64 {
	return m.id
}

func (m *Message) setId(id int64) {
	m.id = id
}

func (m *Message) Account() string {
	return m.acct
}

func (m *Message) setAccount(name string) {
	m.acct = name
}

// Meta-messages

type Done struct {
	Message
}

type Error struct {
	Message
	Error error
}

type Cancelled struct {
	Message
}

type ConnError struct {
	Message
	Error error
}

type Unsupported struct {
	Message
}

// Actions

type Configure struct {
	Message
	Config *config.AccountConfig
}

type Connect struct {
	Message
}

type Reconnect struct {
	Message
}

type Disconnect struct {
	Message
}

type ListDirectories struct {
	Message
}

type OpenDirectory struct {
	Message
	Context   context.Context
	Directory string
	Query     string
	Force     bool
}

type FetchDirectoryContents struct {
	Message
	Context      context.Context
	SortCriteria []*SortCriterion
	Filter       *SearchCriteria
}

type FetchDirectoryThreaded struct {
	Message
	Context       context.Context
	SortCriteria  []*SortCriterion
	Filter        *SearchCriteria
	ThreadContext bool
}

type SearchDirectory struct {
	Message
	Context  context.Context
	Criteria *SearchCriteria
}

type DirectoryThreaded struct {
	Message
	Threads []*Thread
}

type CreateDirectory struct {
	Message
	Directory string
	Quiet     bool
}

type RemoveDirectory struct {
	Message
	Directory string
	Quiet     bool
}

type FetchMessageHeaders struct {
	Message
	Context context.Context
	Uids    []models.UID
}

type FetchFullMessages struct {
	Message
	Uids []models.UID
}

type FetchMessageBodyPart struct {
	Message
	Uid  models.UID
	Part []int
}

type FetchMessageFlags struct {
	Message
	Context context.Context
	Uids    []models.UID
}

type DeleteMessages struct {
	Message
	Uids              []models.UID
	MultiFileStrategy *MultiFileStrategy
}

// Flag messages with different mail types
type FlagMessages struct {
	Message
	Enable bool
	Flags  models.Flags
	Uids   []models.UID
}

type AnsweredMessages struct {
	Message
	Answered bool
	Uids     []models.UID
}

type ForwardedMessages struct {
	Message
	Forwarded bool
	Uids      []models.UID
}

type CopyMessages struct {
	Message
	Destination       string
	Uids              []models.UID
	MultiFileStrategy *MultiFileStrategy
}

type MoveMessages struct {
	Message
	Destination       string
	Uids              []models.UID
	MultiFileStrategy *MultiFileStrategy
}

type AppendMessage struct {
	Message
	Destination string
	Flags       models.Flags
	Date        time.Time
	Reader      io.Reader
	Length      int
}

type CheckMail struct {
	Message
	Directories []string
	Command     string
	Timeout     time.Duration
}

type StartSendingMessage struct {
	Message
	From   *mail.Address
	Rcpts  []*mail.Address
	CopyTo []string
}

// Messages

type Directory struct {
	Message
	Dir *models.Directory
}

type DirectoryInfo struct {
	Message
	Info    *models.DirectoryInfo
	Refetch bool
}

type DirectoryContents struct {
	Message
	Uids []models.UID
}

type SearchResults struct {
	Message
	Uids []models.UID
}

type MessageInfo struct {
	Message
	Info       *models.MessageInfo
	NeedsFlags bool
}

type FullMessage struct {
	Message
	Content *models.FullMessage
}

type MessageBodyPart struct {
	Message
	Part *models.MessageBodyPart
}

type MessagesDeleted struct {
	Message
	Uids []models.UID
}

type MessagesCopied struct {
	Message
	Destination string
	Uids        []models.UID
}

type MessagesMoved struct {
	Message
	Destination string
	Uids        []models.UID
}

type ModifyLabels struct {
	Message
	Uids   []models.UID
	Add    []string
	Remove []string
	Toggle []string
}

type LabelList struct {
	Message
	Labels []string
}

type CheckMailDirectories struct {
	Message
	Directories []string
}

type MessageWriter struct {
	Message
	Writer io.WriteCloser
}
