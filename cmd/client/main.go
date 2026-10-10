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
	defer := bufio.NewScanner(reader)
	if !scanner.Scan() {
	    return resp, fmt.Errorf("no response received from server")
	}

	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
	    return resp, fmr.Errorf("unmarshal response: %w", err
	}

	return resp, nil
}


func runCreate(serverFIFO, topic string){
    // rule: validate topic name format

    if !protocol.ValidateIdentifier(topic) {
        fmt.Fprintf(os.Stderr, "error: invalid topic name: %s\n", topic)
        os.Exit(protocol.ExitGeneral) // 1
    }

    resp. err := sendRequest(serverFIFO, protocol.Request{
        Type: protocol.CmdCreate,
        Topic: topic,
    })
    if err != nil {
        fmt.Fprinf(os.Stderr, "error: communication failure: %v\n", err)
        os.Exit(protocol.ExitIPC) // 3
    }

    if ! resp.Success {
        fmt.Fprintf(os.Stderr, "error: %s\n", resp.Message)
        os.Exit(resp.ExitCode) // 2 already exists
    }

    // output : "topic created"
    fmt.Println("topic created")
    os.Exit(protocol.ExitSucess) // 0
}
