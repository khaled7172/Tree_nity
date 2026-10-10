package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"

	"Tree_nity/internal/ipc"
	"Tree_nity/internal/protocol"
)

var replySeq uint64

func sendRequest(serverFIFO string, req protocol.Request) (protocol.Response, error) {
	var resp protocol.Response

	// generate a unique path for our private reply fifo: /tmp/operator.client.<PID>.<seq>
	seq := atomic.AddUint64(&replySeq, 1)
	replyPath := ipc.ClientReplyEndpoint(os.Getpid(), seq)

	// create the reply FIFO on disk
	if err := ipc.MakeFifo(replyPath); err != nil {
		return resp, fmt.Errorf("create reply fifo: %w", err)
	}
	defer ipc.RemoveFifo(replyPath)

	req.ReplyFIFO = replyPath

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

	reader, err := ipc.OpenReader(replyPath)
	if err != nil {
		return resp, fmt.Errorf("open reply fifo: %w", err)
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		return resp, fmt.Errorf("no response received from server")
	}

	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return resp, fmt.Errorf("unmarshal response: %w", err)
	}

	return resp, nil
}
