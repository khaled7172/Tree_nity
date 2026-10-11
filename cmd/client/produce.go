package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"Tree_nity/internal/ipc"
	"Tree_nity/internal/protocol"
)

func runProduce(serverFIFO, topic string, isRaw bool) {
	// 1. Validate topic name format
	if !protocol.ValidateIdentifier(topic) {
		fmt.Fprintf(os.Stderr, "error: invalid topic name: %s\n", topic)
		os.Exit(protocol.ExitGeneral) // 1
	}

	// 2. Mandatory Handshake: check topic exists with server
	resp, err := sendRequest(serverFIFO, protocol.Request{
		Type:  protocol.CmdProduce,
		Topic: topic,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: communication failure: %v\n", err)
		os.Exit(protocol.ExitIPC) // 3
	}

	if !resp.Success {
		fmt.Fprintf(os.Stderr, "error: %s\n", resp.Message)
		os.Exit(protocol.ExitTopic) // 2 (Topic does not exist)
	}

	// 3. Open server FIFO to stream messages
	writer, err := ipc.OpenWriter(serverFIFO)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to open server pipe: %v\n", err)
		os.Exit(protocol.ExitIPC) // 3
	}
	defer writer.Close()

	sendMsg := func(key, value []byte, rawPayload []byte) error {
		req := protocol.Request{
			Type:    protocol.CmdProduce,
			Topic:   topic,
			Key:     string(key),
			Payload: rawPayload,
		}
		data, err := json.Marshal(req)
		if err != nil {
			return err
		}
		data = append(data, '\n')
		_, err = writer.Write(data)
		return err
	}

	// 4. Stream messages from os.Stdin
	if isRaw {
		// Raw Binary Mode (--raw)
		for {
			key, val, err := protocol.DecodeBinaryProducer(os.Stdin)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break // Clean EOF on record boundary
				}
				if errors.Is(err, protocol.ErrPartialRecord) {
					fmt.Fprintf(os.Stderr, "error: partial record at EOF\n")
					os.Exit(protocol.ExitGeneral) // 1 (Subject Chapter VIII.2)
				}
				fmt.Fprintf(os.Stderr, "error reading binary record: %v\n", err)
				os.Exit(protocol.ExitGeneral) // 1
			}

			var buf bytes.Buffer
			if err := protocol.EncodeBinaryProducer(&buf, key, val); err != nil {
				fmt.Fprintf(os.Stderr, "error encoding record: %v\n", err)
				os.Exit(protocol.ExitGeneral)
			}

			if err := sendMsg(key, val, buf.Bytes()); err != nil {
				fmt.Fprintf(os.Stderr, "error sending message to server: %v\n", err)
				os.Exit(protocol.ExitIPC) // 3
			}
		}
	} else {
		// Text Mode (Default: key:body\n)
		reader := bufio.NewReader(os.Stdin)
		for {
			key, val, err := protocol.DecodeTextMessage(reader)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break // Clean EOF
				}
				fmt.Fprintf(os.Stderr, "error reading text message: %v\n", err)
				os.Exit(protocol.ExitGeneral) // 1
			}

			var buf bytes.Buffer
			if err := protocol.EncodeTextMessage(&buf, key, val); err != nil {
				fmt.Fprintf(os.Stderr, "error encoding message: %v\n", err)
				os.Exit(protocol.ExitGeneral)
			}

			if err := sendMsg(key, val, buf.Bytes()); err != nil {
				fmt.Fprintf(os.Stderr, "error sending message to server: %v\n", err)
				os.Exit(protocol.ExitIPC) // 3
			}
		}
	}

	os.Exit(protocol.ExitSuccess) // 0
}
