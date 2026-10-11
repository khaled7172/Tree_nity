# **************************************************************************** #
#                                                                              #
#                                                         :::      ::::::::    #
#    Makefile                                           :+:      :+:    :+:    #
#                                                     +:+ +:+         +:+      #
#    By: adadra <adadra@student.42beirut.com>       +#+  +:+       +#+         #
#                                                 +#+#+#+#+#+   +#+            #
#    Created: 2026/10/11 19:00:00 by adadra            #+#    #+#              #
#    Updated: 2026/10/11 19:00:00 by adadra           ###   ########.fr        #
#                                                                              #
# **************************************************************************** #

SERVER_BIN := server
CLIENT_BIN := client
GO         := go

.PHONY: all clean re test help jammy-help $(SERVER_BIN) $(CLIENT_BIN)

# Default target: compile both binaries
all: $(SERVER_BIN) $(CLIENT_BIN)

$(SERVER_BIN):
	@echo "Compiling $(SERVER_BIN)..."
	$(GO) build -o $(SERVER_BIN) ./cmd/server

$(CLIENT_BIN):
	@echo "Compiling $(CLIENT_BIN)..."
	$(GO) build -o $(CLIENT_BIN) ./cmd/client

clean:
	@echo "Removing binaries..."
	rm -f $(SERVER_BIN) $(CLIENT_BIN)

re: clean all

test:
	@echo "Running unit tests with race detector..."
	$(GO) test -v -race ./...

# ==============================================================================
# 42 Jammy Helper Binaries Guidelines (Aquarium & Fish)
# ==============================================================================
# The official 42 test suite provides two helper binaries in ./jammy/:
#
# 1. Aquarium Visualizer (Consumer):
#    - Spawns: ./client <ipc> subscribe <topic> <name> --raw
#    - Usage:  ./jammy/aquarium ./client <ipc_identifier> <topic> <name> [flags]
#    - Flags:  -o, --offset <N>   | -p, --prefix <str>
#    - Example:
#        ./jammy/aquarium ./client /tmp/operator.server.12345 ocean nemo
#
# 2. Fish Generator (Producer):
#    - Spawns: ./client <ipc> produce <topic> --raw
#    - Usage:  ./jammy/fish ./client <ipc_identifier> <topic> [flags]
#    - Flags:  --min-delay <ms>   | --max-delay <ms> | --species <name> | --seed <int>
#    - Example:
#        ./jammy/fish ./client /tmp/operator.server.12345 ocean
#
# 3. High-Concurrency Stress Test (100 Concurrent Fish Swarm):
#    - Launch:
#        for i in $$(seq 1 100); do ./jammy/fish ./client /tmp/operator.server.12345 ocean & done
#    - Teardown:
#        pkill -f "jammy/fish"
# ==============================================================================

help:
	@echo "Tree_nity Build & Test Commands:"
	@echo "  make all          - Build server and client binaries"
	@echo "  make clean        - Remove generated binaries"
	@echo "  make re           - Clean and rebuild everything"
	@echo "  make test         - Run all unit tests with race detection"
	@echo "  make jammy-help   - Display full guidelines for ./jammy/aquarium & ./jammy/fish"

jammy-help:
	@echo "======================================================================"
	@echo "          42 Jammy Helper Binaries (Aquarium & Fish) Guide            "
	@echo "======================================================================"
	@echo ""
	@echo "1. Start Server:"
	@echo "   ./server"
	@echo "   (Note the printed IPC path, e.g. /tmp/operator.server.12345)"
	@echo ""
	@echo "2. Create Topic & Launch Aquarium Visualizer:"
	@echo "   ./client /tmp/operator.server.12345 create ocean"
	@echo "   ./jammy/aquarium ./client /tmp/operator.server.12345 ocean nemo"
	@echo ""
	@echo "3. Produce Fish Events:"
	@echo "   ./jammy/fish ./client /tmp/operator.server.12345 ocean"
	@echo ""
	@echo "4. High Concurrency Swarm (100 Producers):"
	@echo "   for i in \$$(seq 1 100); do ./jammy/fish ./client /tmp/operator.server.12345 ocean & done"
	@echo "   To stop swarm: pkill -f \"jammy/fish\""
	@echo "======================================================================"
