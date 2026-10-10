package main

import (
	"fmt"
	"os"
	"strconv"

	"Tree_nity/internal/ipc"
	"Tree_nity/internal/protocol"
)

func main() {
	// Usage: ./client <ipc_identifier> <command> [args...]
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <ipc_identifier> <create|list|info|produce|subscribe> [args...]\n", os.Args[0])
		os.Exit(protocol.ExitGeneral) // Exit 1
	}

	serverFIFO := os.Args[1]
	command := os.Args[2]

	// Verify that the provided IPC identifier exists and is a valid FIFO pipe
	if !ipc.IsFifo(serverFIFO) {
		fmt.Fprintf(os.Stderr, "error: invalid ipc identifier: %s is not a FIFO\n", serverFIFO)
		os.Exit(protocol.ExitGeneral) // Exit 1 (Highest precedence)
	}

	switch command {
	case "create":
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "usage: %s <ipc> create <topic_name>\n", os.Args[0])
			os.Exit(protocol.ExitGeneral)
		}
		runCreate(serverFIFO, os.Args[3])

	case "list":
		runList(serverFIFO)

	case "info":
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "usage: %s <ipc> info <subscriber_name>\n", os.Args[0])
			os.Exit(protocol.ExitGeneral)
		}
		runInfo(serverFIFO, os.Args[3])

	case "produce":
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "usage: %s <ipc> produce <topic_name> [--raw]\n", os.Args[0])
			os.Exit(protocol.ExitGeneral)
		}
		topic := os.Args[3]
		isRaw := false
		for _, arg := range os.Args[4:] {
			if arg == "--raw" {
				isRaw = true
			}
		}
		runProduce(serverFIFO, topic, isRaw)

	case "subscribe":
		if len(os.Args) < 5 {
			fmt.Fprintf(os.Stderr, "usage: %s <ipc> subscribe <topic_name> <subscriber_name> [--prefix <prefix>] [--offset <offset>] [--raw]\n", os.Args[0])
			os.Exit(protocol.ExitGeneral)
		}
		topic := os.Args[3]
		clientID := os.Args[4]

		var prefix string
		var offset *uint32
		isRaw := false

		// Parse flags: --prefix, --offset, --raw
		args := os.Args[5:]
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--prefix":
				if i+1 < len(args) {
					prefix = args[i+1]
					i++
				}
			case "--offset":
				if i+1 < len(args) {
					val, err := strconv.ParseUint(args[i+1], 10, 32)
					if err != nil {
						fmt.Fprintf(os.Stderr, "error: invalid offset value: %s\n", args[i+1])
						os.Exit(protocol.ExitGeneral)
					}
					uVal := uint32(val)
					offset = &uVal
					i++
				}
			case "--raw":
				isRaw = true
			default:
				fmt.Fprintf(os.Stderr, "error: unknown flag %q\n", args[i])
				os.Exit(protocol.ExitGeneral)
			}
		}

		runSubscribe(serverFIFO, topic, clientID, prefix, offset, isRaw)

	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n", command)
		os.Exit(protocol.ExitGeneral)
	}
}
