package main

import (
	"encoding/json"
	"fmt"
	"os"

	"Tree_nity/internal/protocol"
)

func runCreate(serverFIFO, topic string) {
	if !protocol.ValidateIdentifier(topic) {
		fmt.Fprintf(os.Stderr, "error: invalid topic name: %s\n", topic)
		os.Exit(protocol.ExitGeneral) // 1
	}

	resp, err := sendRequest(serverFIFO, protocol.Request{
		Type:  protocol.CmdCreate,
		Topic: topic,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: communication failure: %v\n", err)
		os.Exit(protocol.ExitIPC) // 3
	}

	if !resp.Success {
		fmt.Fprintf(os.Stderr, "error: %s\n", resp.Message)
		os.Exit(resp.ExitCode) // 2 already exists
	}

	fmt.Println("topic created")
	os.Exit(protocol.ExitSuccess) // 0
}

func runList(serverFIFO string) {
	resp, err := sendRequest(serverFIFO, protocol.Request{
		Type: protocol.CmdList,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: communication failure: %v\n", err)
		os.Exit(protocol.ExitIPC) // 3
	}

	if !resp.Success {
		fmt.Fprintf(os.Stderr, "error: %s\n", resp.Message)
		os.Exit(resp.ExitCode)
	}

	for i, t := range resp.Topics {
		if i > 0 {
			fmt.Print(",")
		}
		fmt.Print(t)
	}
	fmt.Println()
	os.Exit(protocol.ExitSuccess) // 0
}

func runInfo(serverFIFO, clientID string) {
	if !protocol.ValidateIdentifier(clientID) {
		fmt.Fprintf(os.Stderr, "error: invalid subscriber name: %s\n", clientID)
		os.Exit(protocol.ExitGeneral) // 1
	}

	resp, err := sendRequest(serverFIFO, protocol.Request{
		Type:     protocol.CmdInfo,
		ClientID: clientID,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: communication failure: %v\n", err)
		os.Exit(protocol.ExitIPC) // 3
	}

	if !resp.Success || resp.Metadata == nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", resp.Message)
		os.Exit(resp.ExitCode) // 2
	}

	data, err := json.Marshal(resp.Metadata)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to format info: %v\n", err)
		os.Exit(protocol.ExitGeneral)
	}
	fmt.Println(string(data))
	os.Exit(protocol.ExitSuccess) // 0
}
