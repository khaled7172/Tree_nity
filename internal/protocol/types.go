package protocol

import (
	"regexp"
)

const (
	ExitSuccess = 0
	ExitGeneral = 1
	ExitTopic   = 2
	ExitIPC     = 3
)

const (
	MaxMessagePayload = 1024
)

var IdentifierRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,32}$`)

func ValidateIdentifier(id string) bool {
	return IdentifierRegex.MatchString(id)
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
	Type      CommandType `json:"type"`
	Topic     string      `json:"topic,omitempty"`
	ClientID  string      `json:"client_id,omitempty"`
	Prefix    string      `json:"prefix,omitempty"`
	Offset    *uint32     `json:"offset,omitempty"`
	ReplyFIFO string      `json:"reply_fifo,omitempty"`
	Key       string      `json:"key,omitempty"`
	Payload   []byte      `json:"payload,omitempty"`
}

type Response struct {
	Success  bool        `json:"success"`
	ExitCode int         `json:"exit_code"`
	Message  string      `json:"message,omitempty"`
	Topics   []string    `json:"topics,omitempty"`
	Metadata *ClientInfo `json:"metadata,omitempty"`
}

type ClientInfo struct {
	Client string `json:"client"`
	Topic  string `json:"topic"`
	Offset uint32 `json:"offset"`
	Prefix string `json:"prefix"`
	IPC    string `json:"ipc"`
}
