package main

import (
	"Tree_nity/internal/ipc"
	"Tree_nity/internal/protocol"
	"encoding/json"
	"fmt"
	"os"
)

var replySeq uint64

func sendRequest(serverFIFO string, req protocol.Request) (protocol.Response, error) {
	var resp protocol.Response

	// generate a unique path for out private reply fifo: /tmp/operator.client.<PID>.<seq>

	seq := atomic.AddUint(&replySeq, 1)
	replyPath := ipc.ClientReplyEndpoint(os.Getpid(), seq)

	// create the reply FIFO on disk
	if err := ipc.MakeFifo(replyPath); err != nil {
		return resp, fmt.Errorf("create reply fifo: %w", err)
	}
	// guarantee that this pipe is deleted from disk when this function returns
	defer ipc.RemoveFifo(replyPath)

	// attach our reply pipe to the request so the server knows where to reply
	req.ReplyFIFO = replyPath

	// convert the request struct into a single JSON line ending in \n
	data, err := json.Marshal(req)
	if err != nil {
		return resp, fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')
	writer, err := ipc.OpenWriter(serverFIFO)
	if err != nil {
		return resp, fmt.Errorf("open server fifo: %w", err)
	}
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return resp, fmt.Errorf("write to server fifo: %w", err)
	}
	_ = writer.Close()

	// open our private reply FIFO to catch the sever's response
	reader, err := ipc.OpenReader(replyPath)
	if err != nil {
		return resp, fmt.Errorf("open reply fifo: %w", err)
	}
}
