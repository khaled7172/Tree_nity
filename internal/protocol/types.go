package protocol

import (
	"regexp"
)

const (
	ExitSuccess = 0 // sucess
	ExitGeneral = 1 // invalid CLI arguments, regex mismatch, EOF partial record
	ExitTopic   = 2 // Topic/client error: topic not found already exists duplicate subsciber
	ExitIPC     = 3 // IPC communication error: server disconnection pipe failure
)

const (
	//  subject specifies max 1024 bytes for key + body (excluding metadata)
	MaxMessagePayload = 1024
	// names must be 1 to 32 characters
	MaxIdentifiernLen = 32
)

// matches client and topic names
var IdentifierRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,32}$`)

// checks if the topic name or client ID matches the required pattern.
func ValidateIdentifier(id string) bool {
	return IdentifierRegex.MatchString(id)
}

// Message represents a single message stored or transmitted in the message queue
type Message struct {
	Offset uint32
	Key    []byte
	Value  []byte
}

type CommandType string

const (
	CmdCreate    CommandType = "create"
	CmdList      CommandType = "list"
	CmdProduce   CommandType = "produce"
	CmdSubscribe CommandType = "subscribe"
	CmdInfo      CommandType = "info"
	CmdDrainAck  CommandType = "drain_ack"
)

type Request struct {
	Type          CommandType `json:"type"`
	Topic         string      `json:"topic,omitempty"`
	ClientID      string      `json:"client_id,omitempty"`
	Prefix        string      `json:"prefix,omitempty"`
	Offset        *uint32     `json:"offset,omitempty"`
	Raw           bool        `json:"raw,omitempty"`
	ReplyFIFO     string      `json:"reply_fifo,omitempty"`     // Ephemeral pipe for single response
	DedicatedFIFO string      `json:"dedicated_fifo,omitempty"` // Dedicated pipe for subscribed messages
}

// Response is sent by the Server back to the Client's reply pipe
type Response struct {
	Success  bool        `json:"success"`
	ExitCode int         `json:"exit_code"` // 0, 1, 2 or 3
	Message  string      `json:"message,omitempty"`
	Topics   []string    `json:"topics,omitempty"`   // For CmdList
	Metadata *ClientInfo `json:"metadata,omitempty"` // for cmdinfo
}

// represents the JSON structure output by: ./client <ipc> info <subscriber_name>
// output example:
// {"client":"client0","topic":"user_events","offset":4,"prefix":"user.create","ipc":"/tmp/..."}

type ClientInfo struct {
	Client string `json:"client"`
	Topic  string `json:"topic"`
	Offset uint32 `json:"offset"`
	Prefix string `json:"prefix"`
	IPC    string `json:"ipc"`
}
