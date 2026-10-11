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
		os.Exit(protocol.ExitGeneral) // Exit 1
	}

	// 2. Mandatory handshake: verify topic exists
	resp, err := sendRequest(serverFIFO, protocol.Request{
		Type:  protocol.CmdProduce,
		Topic: topic,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: communication failure: %v\n", err)
		os.Exit(protocol.ExitIPC) // Exit 3
	}
	if !resp.Success {
		fmt.Fprintf(os.Stderr, "error: %s\n", resp.Message)
		os.Exit(protocol.ExitTopic) // Exit 2 (Topic does not exist)
	}

	// 3. Open server pipe for message streaming
	writer, err := ipc.OpenWriter(serverFIFO)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to open server pipe: %v\n", err)
		os.Exit(protocol.ExitIPC) // Exit 3
	}
	defer writer.Close()

	// 4. Stream input based on mode
	if isRaw {
		streamRaw(writer, topic)
	} else {
		streamText(writer, topic)
	}

	os.Exit(protocol.ExitSuccess) // Exit 0
}

func sendProduceMessage(w io.Writer, topic, key string, payload []byte) error {
	req := protocol.Request{
		Type:    protocol.CmdProduce,
		Topic:   topic,
		Key:     key,
		Payload: payload,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}

func streamRaw(w io.Writer, topic string) {
	for {
		key, val, err := protocol.DecodeBinaryProducer(os.Stdin)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return // Clean EOF on record boundary
			}
			if errors.Is(err, protocol.ErrPartialRecord) {
				fmt.Fprintf(os.Stderr, "error: partial record at EOF\n")
				os.Exit(protocol.ExitGeneral) // Exit 1 (Subject Chapter VIII.2)
			}
			fmt.Fprintf(os.Stderr, "error reading binary record: %v\n", err)
			os.Exit(protocol.ExitGeneral) // Exit 1
		}

		var buf bytes.Buffer
		if err := protocol.EncodeBinaryProducer(&buf, key, val); err != nil {
			fmt.Fprintf(os.Stderr, "error encoding record: %v\n", err)
			os.Exit(protocol.ExitGeneral)
		}

		if err := sendProduceMessage(w, topic, string(key), buf.Bytes()); err != nil {
			fmt.Fprintf(os.Stderr, "error sending to server: %v\n", err)
			os.Exit(protocol.ExitIPC) // Exit 3
		}
	}
}

func streamText(w io.Writer, topic string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		key, val, err := protocol.DecodeTextMessage(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return // Clean EOF
			}
			fmt.Fprintf(os.Stderr, "error reading text message: %v\n", err)
			os.Exit(protocol.ExitGeneral) // Exit 1
		}

		var buf bytes.Buffer
		if err := protocol.EncodeTextMessage(&buf, key, val); err != nil {
			fmt.Fprintf(os.Stderr, "error encoding message: %v\n", err)
			os.Exit(protocol.ExitGeneral)
		}

		if err := sendProduceMessage(w, topic, string(key), buf.Bytes()); err != nil {
			fmt.Fprintf(os.Stderr, "error sending to server: %v\n", err)
			os.Exit(protocol.ExitIPC) // Exit 3
		}
	}
}
