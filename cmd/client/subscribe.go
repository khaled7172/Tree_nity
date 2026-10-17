package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"Tree_nity/internal/ipc"
	"Tree_nity/internal/protocol"
)

const ShutdownSentinel = "__TREENITY_SHUTDOWN__\n"

func runSubscribe(serverFIFO, topic, clientID, prefix string, offset *uint32, isRaw bool) {
	if !protocol.ValidateIdentifier(topic) || !protocol.ValidateIdentifier(clientID) {
		fmt.Fprintf(os.Stderr, "error: invalid topic or subscriber name\n")
		os.Exit(protocol.ExitGeneral)
	}

	req := protocol.Request{
		Type:     protocol.CmdSubscribe,
		Topic:    topic,
		ClientID: clientID,
		Prefix:   prefix,
		Offset:   offset,
	}

	resp, err := sendRequest(serverFIFO, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: communication failure: %v\n", err)
		os.Exit(protocol.ExitIPC)
	}

	if !resp.Success {
		fmt.Fprintf(os.Stderr, "error: %s\n", resp.Message)
		os.Exit(resp.ExitCode)
	}

	currentOffset := resp.Metadata.Offset
	if offset != nil {
		currentOffset = *offset
	}

	fmt.Printf("subscribed to %s\n", topic)

	reader, err := ipc.OpenReader(resp.Metadata.IPC)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to open consumer channel: %v\n", err)
		os.Exit(protocol.ExitIPC)
	}
	defer reader.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	cleanupAndExit := func() {
		_, _ = sendRequest(serverFIFO, protocol.Request{
			Type:     protocol.CmdDrainAck,
			ClientID: clientID,
		})
		os.Exit(protocol.ExitSuccess)
	}

	go func() {
		<-sigChan
		cleanupAndExit()
	}()

	bufReader := bufio.NewReader(reader)
	if isRaw {
		for {
			peekBytes, err := bufReader.Peek(len(ShutdownSentinel))
			if err == nil && bytes.Equal(peekBytes, []byte(ShutdownSentinel)) {
				cleanupAndExit()
			}

			key, val, err := protocol.DecodeBinaryProducer(bufReader)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				break
			}

			if err := protocol.EncodeBinaryConsumer(os.Stdout, currentOffset, key, val); err != nil {
				break
			}
			currentOffset++
		}
	} else {
		for {
			line, err := bufReader.ReadBytes('\n')
			if err != nil {
				break
			}

			if bytes.Equal(line, []byte(ShutdownSentinel)) {
				cleanupAndExit()
			}

			_, _ = os.Stdout.Write(line)
			currentOffset++
		}
	}

	os.Exit(protocol.ExitSuccess)
}
