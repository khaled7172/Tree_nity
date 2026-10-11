package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"Tree_nity/internal/ipc"
	"Tree_nity/internal/protocol"
)

const ShutdownSentinel = "__TREENITY_SHUTDOWN__\n"

func extractServerPID(serverFIFO string) (int, error) {
	base := filepath.Base(serverFIFO)
	var pid int
	n, err := fmt.Sscanf(base, "operator.server.%d", &pid)
	if err != nil || n != 1 {
		return 0, fmt.Errorf("unable to extract server PID from %s", serverFIFO)
	}
	return pid, nil
}

func runSubscribe(serverFIFO, topic, clientID, prefix string, offset *uint32, isRaw bool) {
	// 1. Validate topic and client ID format
	if !protocol.ValidateIdentifier(topic) || !protocol.ValidateIdentifier(clientID) {
		fmt.Fprintf(os.Stderr, "error: invalid topic or subscriber name\n")
		os.Exit(protocol.ExitGeneral) // 1
	}

	serverPID, err := extractServerPID(serverFIFO)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid ipc identifier: %v\n", err)
		os.Exit(protocol.ExitGeneral) // 1
	}

	// 2. Subscribe registration request
	req := protocol.Request{
		Type:     protocol.CmdSubscribe,
		Topic:    topic,
		ClientID: clientID,
		Prefix:   prefix,
		Offset:   offset,
		Raw:      isRaw,
	}

	resp, err := sendRequest(serverFIFO, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: communication failure: %v\n", err)
		os.Exit(protocol.ExitIPC) // 3
	}

	if !resp.Success {
		fmt.Fprintf(os.Stderr, "error: %s\n", resp.Message)
		os.Exit(resp.ExitCode) // 2 for topic/client error (e.g. topic not found, duplicate subscriber)
	}

	// Determine starting offset
	currentOffset := uint32(0)
	if offset != nil {
		currentOffset = *offset
	} else if resp.Metadata != nil {
		currentOffset = resp.Metadata.Offset
	}

	// Output (stdout) confirmation mandated by subject: "subscribed to <topic>"
	fmt.Printf("subscribed to %s\n", topic)

	// 3. Open our dedicated consumer FIFO
	consumerPath := ipc.ConsumerEndpoint(serverPID, clientID)
	reader, err := ipc.OpenReader(consumerPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to open consumer channel: %v\n", err)
		os.Exit(protocol.ExitIPC) // 3
	}
	defer reader.Close()

	// 4. Graceful signal handling (Ctrl+C / SIGINT / SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	cleanupAndExit := func() {
		// Send drain acknowledgment to server so server cleans up and persists offset
		_, _ = sendRequest(serverFIFO, protocol.Request{
			Type:     protocol.CmdDrainAck,
			ClientID: clientID,
			Offset:   &currentOffset,
		})
		os.Exit(protocol.ExitSuccess) // 0
	}

	go func() {
		<-sigChan
		cleanupAndExit()
	}()

	// 5. Message ingestion loop
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
				// If pipe closed without sentinel, exit 0 on clean disconnect
				break
			}

			// Check for shutdown sentinel pushed by server
			if bytes.Equal(line, []byte(ShutdownSentinel)) {
				cleanupAndExit()
			}

			// Output matching message directly to stdout
			_, _ = os.Stdout.Write(line)
			currentOffset++
		}
	}

	os.Exit(protocol.ExitSuccess) // 0
}
