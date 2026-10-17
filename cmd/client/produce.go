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
	if !protocol.ValidateIdentifier(topic) {
		fmt.Fprintf(os.Stderr, "error: invalid topic name: %s\n", topic)
		os.Exit(protocol.ExitGeneral)
	}

	resp, err := sendRequest(serverFIFO, protocol.Request{
		Type:  protocol.CmdProduce,
		Topic: topic,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: communication failure: %v\n", err)
		os.Exit(protocol.ExitIPC)
	}
	if !resp.Success {
		fmt.Fprintf(os.Stderr, "error: %s\n", resp.Message)
		os.Exit(protocol.ExitTopic)
	}

	writer, err := ipc.OpenWriter(serverFIFO)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to open server pipe: %v\n", err)
		os.Exit(protocol.ExitIPC)
	}
	defer writer.Close()

	if isRaw {
		streamRaw(writer, topic)
	} else {
		streamText(writer, topic)
	}

	os.Exit(protocol.ExitSuccess)
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
				return
			}
			if errors.Is(err, protocol.ErrPartialRecord) {
				fmt.Fprintf(os.Stderr, "error: partial record at EOF\n")
				os.Exit(protocol.ExitGeneral)
			}
			fmt.Fprintf(os.Stderr, "error reading binary record: %v\n", err)
			os.Exit(protocol.ExitGeneral)
		}

		var buf bytes.Buffer
		if err := protocol.EncodeBinaryProducer(&buf, key, val); err != nil {
			fmt.Fprintf(os.Stderr, "error encoding record: %v\n", err)
			os.Exit(protocol.ExitGeneral)
		}

		if err := sendProduceMessage(w, topic, string(key), buf.Bytes()); err != nil {
			fmt.Fprintf(os.Stderr, "error sending to server: %v\n", err)
			os.Exit(protocol.ExitIPC)
		}
	}
}

func streamText(w io.Writer, topic string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		key, val, err := protocol.DecodeTextMessage(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			fmt.Fprintf(os.Stderr, "error reading text message: %v\n", err)
			os.Exit(protocol.ExitGeneral)
		}

		var buf bytes.Buffer
		if err := protocol.EncodeTextMessage(&buf, key, val); err != nil {
			fmt.Fprintf(os.Stderr, "error encoding message: %v\n", err)
			os.Exit(protocol.ExitGeneral)
		}

		if err := sendProduceMessage(w, topic, string(key), buf.Bytes()); err != nil {
			fmt.Fprintf(os.Stderr, "error sending to server: %v\n", err)
			os.Exit(protocol.ExitIPC)
		}
	}
}
