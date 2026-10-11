*This project has been created as part of the 42 curriculum by adadra, cbahry, khhammou.*

# Tree_nity: In-Memory Message Queue System

An asynchronous, high-concurrency pub/sub message broker and client CLI implemented in Go using POSIX Named Pipes (FIFOs).

---

## 1. Description

**Tree_nity** is an event-driven message queue system built from scratch without external dependencies or brokers. It decouples event producers from consumers through named communication channels called **topics**.

Key capabilities:
- **Asynchronous Pub/Sub:** Decouples producer publishing from consumer message processing.
- **Dedicated Topic Workers:** Independent worker goroutines isolate topic throughput and prevent noisy-neighbor stalls.
- **Prefix-Based Routing:** Consumers can filter topic events using key prefixes (e.g. `order.created`, `sensor.temp`).
- **Offset Tracking & Resumption:** Consumers commit their next expected offset upon graceful shutdown and resume consumption seamlessly across reconnects.
- **Dual Wire Formats:** Full support for both human-readable text streams (`key:body\n`) and high-performance Little-Endian raw binary records.
- **Zero-Dependency IPC:** POSIX Named Pipes (FIFOs) provide standard OS-level IPC without external daemons, sockets, or CGo.

---

## 2. Instructions

### 2.1 Compilation
The root `Makefile` provides standardized build targets:

```bash
# Build both server and client binaries
make all

# Clean compiled binaries
make clean

# Recompile from scratch
make re

# Run all unit tests with Go's race detector
make test

# Display helper documentation for jammy tester tools
make jammy-help
```

---

### 2.2 Starting the Server
Start the message broker in a terminal:
```bash
./server
```
On startup, the server creates its primary communication FIFO and prints its endpoint to stdout:
```
/tmp/operator.server.12345
```
*(Keep this terminal open. Export the path in other terminals: `export IPC=/tmp/operator.server.12345`)*

---

### 2.3 Client CLI Operations

#### Management Commands
```bash
# 1. Create a topic
./client $IPC create ocean
# Output: topic created (exit 0)

# 2. List all topics
./client $IPC list
# Output: ocean (exit 0)

# 3. Query subscriber metadata
./client $IPC info nemo
# Output: JSON object with client, topic, offset, prefix, and ipc path
```

#### Producer Mode
Producers stream messages from `stdin` to a target topic. Messages follow `key:body\n` format:
```bash
# Single message via echo
echo "fish.bass.spawn:{\"x\":100,\"y\":200}" | ./client $IPC produce ocean

# Batch messages via printf
printf "metric.cpu:75\nmetric.ram:60\n" | ./client $IPC produce metrics

# Raw binary mode (--raw)
./client $IPC produce ocean --raw < binary_events.bin
```

#### Consumer Mode (Subscribe)
Consumers establish a dedicated streaming FIFO with the server and receive matching events:
```bash
# 1. Standard subscription (receives all messages on the topic)
./client $IPC subscribe ocean nemo

# 2. Prefix filtering (--prefix)
# Only receives messages whose keys start with "fish.bass"
./client $IPC subscribe ocean nemo --prefix fish.bass

# 3. Offset replay (--offset)
# Replays historical messages starting from offset 0
./client $IPC subscribe ocean auditor --offset 0

# 4. Raw binary consumer output (--raw)
./client $IPC subscribe ocean visualizer --raw
```
Press `Ctrl+C` (`SIGINT`) or send `SIGTERM` to disconnect a subscriber gracefully. The subscriber sends a drain acknowledgment (`CmdDrainAck`), committing its next expected offset to the server.

---

### 2.4 Official 42 Tester Binaries (`jammy`)

The subject provides two helper binaries in the `./jammy/` folder:

1. **`jammy/aquarium` (2D Visualizer / Consumer)**
   - Connects to a topic via `./client <ipc> subscribe <topic> <name> --raw`.
   - Renders incoming fish events in a real-time graphical window.
   ```bash
   ./jammy/aquarium ./client $IPC ocean nemo
   ```

2. **`jammy/fish` (Event Generator / Producer)**
   - Connects to a topic via `./client <ipc> produce <topic> --raw`.
   - Generates random swim, spawn, and bubble events.
   ```bash
   ./jammy/fish ./client $IPC ocean
   ```

3. **High-Concurrency Swarm Test (100 Concurrent Fish Producers)**
   ```bash
   for i in $(seq 1 100); do
       ./jammy/fish ./client $IPC ocean &
   done
   
   # Stop the swarm:
   pkill -f "jammy/fish"
   ```

---

## 3. Architecture & Design Choices

```
               +-------------------------------------------------------+
               |                  Tree_nity Broker                     |
               |                                                       |
               |   Primary FIFO: /tmp/operator.server.<PID>            |
               |                     |                                 |
+------------+ |            [ Dispatch Loop ]                          |
|  Producer  | |                     |                                 |
|  (Client)  |==> [produce] ==> [Topic Worker] === (Publish) ===+      |
+------------+ |                     |                          |      |
               |               [Topic Store] (Ring/Append)      |      |
               |                     |                          |      |
               |   Dedicated FIFO:   | (Match & Push)           |      |
               |   /tmp/operator.server.<PID>.<clientID>        |      |
               |                     |                          |      |
+------------+ |                     v                          v      |
|  Consumer  |<=================================================+      |
|  (Client)  |                                                         |
+------------+ +-------------------------------------------------------+
```

### 3.1 Concurrency Model
- **Worker-per-Topic:** Each topic is managed by an independent `TopicWorker` goroutine with an inbound channel buffer of 1024 messages. This ensures that heavy load on one topic does not block operations on other topics.
- **Thread Safety:** State transitions, client registries, and message appends are protected by fine-grained `sync.RWMutex` locks, ensuring non-blocking concurrent reads and safe mutual exclusion during writes.
- **Graceful Drain & Sentinel:** When the server shuts down, it broadcasts a `ShutdownSentinel` (`__TREENITY_SHUTDOWN__\n`) across all active consumer pipes, allows up to 5 seconds for consumers to acknowledge and persist their offsets, and cleans up all OS FIFOs.

### 3.2 IPC Mechanism (POSIX Named Pipes)
- **Why Named Pipes (FIFOs):** POSIX FIFOs provide reliable, unidirectional byte streams with kernel-enforced FIFO ordering, zero network overhead, and zero CGo dependencies. They can be inspected directly in `/tmp` during debugging.
- **Ephemeral Reply Channels:** Management requests (`create`, `list`, `info`) pass an ephemeral reply pipe path (`/tmp/operator.client.<PID>.<seq>`) in the request JSON. The server writes the response and the client removes the pipe immediately.
- **Dedicated Consumer Channels:** When a client subscribes, the server creates a dedicated consumer FIFO (`/tmp/operator.server.<PID>.<clientID>`), isolating each subscriber's stream.

### 3.3 Custom In-Memory Data Structures
As mandated by 42 constraints, the project avoids third-party packages and implements custom data structures:

1. **Custom Hash Map (`internal/hashmap`):**
   - Implements collision resolution via **separate chaining** with singly-linked node buckets.
   - Uses the **FNV-1a 64-bit hashing algorithm** for uniform distribution.
   - Features **dynamic resizing**: doubles capacity when the load factor exceeds `0.75`.
   - Thread-safe operations protected by `sync.RWMutex`.

2. **Custom Prefix Trie (`internal/trie`):**
   - Hierarchical radix trie for string prefix routing.
   - Supports wildcard subscriptions (`""`) matching all messages on a topic.
   - Features **reverse ghost-branch pruning**: unregistering a subscriber recursively prunes empty intermediate nodes, preventing memory leaks during long-running sessions.

### 3.4 Wire Protocol & Message Codecs (`internal/protocol`)
- **Text Mode:** `key:value\n` with max payload length of 1024 bytes. Missing colon separator returns a format error.
- **Raw Binary Mode (Little-Endian):**
  - Producer Input: `[keysize:int32][key:bytes][valuesize:int32][value:bytes]`
  - Consumer Output: `[offset:int32][keysize:int32][key:bytes][valuesize:int32][value:bytes]`
  - Enforces strict record boundary alignment: a stream terminating mid-record returns `ErrPartialRecord` (Exit Code 1).

---

## 4. Exit Codes Reference

| Exit Code | Classification | Trigger Conditions |
|:---:|---|---|
| **`0`** | **Success** | Successful command completion, graceful consumer disconnect on SIGINT/SIGTERM |
| **`1`** | **General / Protocol Error** | Invalid identifier, payload > 1024 bytes, partial binary record at EOF, nonexistent server FIFO |
| **`2`** | **Topic / Subscriber Error** | Topic does not exist on produce/subscribe, topic already exists, duplicate subscriber name running simultaneously |
| **`3`** | **IPC / Communication Error** | Server FIFO write failure, consumer pipe broken, IPC communication timeout |

---

## 5. Resources & AI Disclosure

### References
- [POSIX FIFO / Named Pipes Manual](https://man7.org/linux/man-pages/man7/fifo.7.html)
- [Go Concurrency Patterns: Pipelines and cancellation](https://go.dev/blog/pipelines)
- [FNV Hash Algorithm Specification](https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function)
- [Conventional Commits v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/)

### AI Tool Usage Disclosure
In accordance with 42 academic guidelines, AI assistance (Claude / DeepMind / Antigravity pair programmer) was utilized during the development of this project:
- **Tasks Assigned:** Code architecture discussion, refactoring modular client subcommands, protocol wire codec edge-case handling (partial EOF detection), troubleshooting binary little-endian endianness, and drafting comprehensive technical documentation.
- **Verification:** All generated code was manually reviewed, compiled with `-race`, verified against `gitinette` for workflow guidelines, and tested against the official `jammy/aquarium` and `jammy/fish` tester binaries.

---

## 6. Authors

This project was built by:
- **`adadra`** (Adam Adra) - Client CLI, Wire Codecs, Makefile, Documentation & Integration
- **`cbahry`** (Charbel Bahry) - Server Dispatcher, IPC Channel Layer & Graceful Shutdown
- **`khhammou`** - Custom Hashmap, Topic Buffer Store & Prefix Trie
