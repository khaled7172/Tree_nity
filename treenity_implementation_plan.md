# Treenity: Comprehensive Architecture & Implementation Plan

> *A concurrent, event-driven message queue system built with Go 1.27 using pure POSIX Named Pipes (FIFOs).*

---

## 1. Executive Summary & Design Decisions

### 1.1 Why Go 1.27 and Pure Go Named Pipes (FIFOs)?
The subject mandates choosing between System V Message Queues, POSIX Message Queues, or Named Pipes (FIFOs).
* **POSIX / System V Message Queues in Go** require either `CGo` (calling C headers `mqueue.h` / `sys/msg.h`) or raw `syscall.Syscall` wrappers, which introduce build complexity, cross-compilation hurdles, runtime overhead, and violate the core project requirement: *"the approach should be simple with no fancy overengineering and trying to avoid CGo"*.
* **Named Pipes (FIFOs)** in Linux are first-class citizens. They can be created via standard Go `golang.org/x/sys/unix.Mkfifo` or `syscall.Mkfifo(path, mode)` (standard library, zero external dependencies, **zero CGo**).
* In Go, reading and writing to a FIFO integrates seamlessly with standard library interfaces: `os.OpenFile`, `io.Reader`, `io.Writer`, `bufio.Reader`, `bufio.Writer`, and `encoding/binary`.
* FIFOs are visible in the filesystem (e.g. under `/tmp/`), making inspection, debugging, testing, and lifecycle cleanup straightforward and transparent.

### 1.2 IPC Architecture & Channel Topology
To satisfy the requirements of bidirectional communication, multiple server instances, and dedicated consumer channels:

1. **Main Server Endpoint**:
   - Named: `/tmp/treenity.server.<PID>` (or `/tmp/operator.server.<PID>`).
   - Created on server startup using `syscall.Mkfifo(endpoint, 0660)`.
   - Server prints this exact path to `stdout` upon startup.
   - Used by clients for initial management requests (`create`, `list`, `info`, and consumer registration).

2. **Bidirectional Management Channel (Client Request / Response)**:
   - Because a FIFO is unidirectional, client management operations create an ephemeral response FIFO: `/tmp/treenity.client.<PID>.<random/seq>`.
   - The client writes its command request containing `reply_fifo_path` to the server's main FIFO.
   - The server writes the response back to the client's reply FIFO and closes it.
   - The client reads the response, unlinks its reply FIFO, and prints the result to `stdout`.

3. **Producer Channel**:
   - A producer verifies topic existence via the main endpoint, then streams messages into the server's topic ingest channel (or directly via a dedicated topic pipe / main pipe with message framing).

4. **Dedicated Consumer Channels**:
   - Each subscribed consumer has its own dedicated FIFO: `/tmp/treenity.server.<PID>.<client_id>`.
   - Server pushes matched messages (based on key prefix) sequentially to this dedicated channel.
   - Consumer reads each message, outputs it to `stdout`, and commits the acknowledged offset back to the server.

5. **Graceful Shutdown & Sentinel Waking**:
   - Subject requirement: *A plain `close()` on the server side will not necessarily wake up a client blocked reading it. On shutdown, the server must get every consumer to exit cleanly (exit code 0).*
   - **Mechanism**: The server sends a **Sentinel Message** (e.g. `Type: SHUTDOWN_SENTINEL` or an empty record with a designated control flag) on each consumer's dedicated channel before closing it. Upon reading this sentinel, the consumer exits cleanly with status code `0`.
   - A 5-second drain timer protects against hung clients.

---

## 2. Project Directory Structure

A clean, idiomatic, standard Go project layout (`cmd/` for executables, `internal/` for private packages):

```
treenity/
├── Makefile                          # all, clean, re, test
├── README.md                         # Required 42 README template & documentation
├── go.mod                            # Go module definition (go 1.27)
├── cmd/
│   ├── server/
│   │   └── main.go                   # Server executable entry point
│   └── client/
│       └── main.go                   # Client executable entry point
├── internal/
│   ├── hashmap/                      # CUSTOM HASHMAP (Mandatory from scratch)
│   │   ├── hashmap.go                # Custom Hashmap with collision resolution
│   │   ├── hashmap_test.go           # Hashmap unit tests
│   │   └── client_metadata.go        # Client metadata struct & methods
│   ├── trie/                         # PREFIX MATCHING DATA STRUCTURE
│   │   ├── trie.go                   # Trie / Radix tree implementation
│   │   └── trie_test.go              # Comprehensive prefix matching tests
│   ├── ipc/                          # PURE GO NAMED PIPE SUBSYSTEM
│   │   ├── fifo.go                   # Mkfifo, open, read/write, unlink helpers
│   │   ├── listener.go               # Server incoming request reader
│   │   └── client_channel.go         # Dedicated consumer channel writer/reader
│   ├── protocol/                     # MESSAGE ENCODING & PROTOCOL
│   │   ├── wire.go                   # Binary (little-endian) & Text serialization
│   │   ├── types.go                  # Command and Message structs
│   │   ├── validator.go              # Regex validation for ClientID & Topic
│   │   └── errors.go                 # Error definitions & Exit codes (1, 2, 3)
│   ├── storage/                      # IN-MEMORY MESSAGE STORE
│   │   ├── topic.go                  # In-memory slice/ring buffer & offset tracking
│   │   └── topic_manager.go          # Thread-safe registry of all active topics
│   └── engine/                       # SERVER LOGIC & TOPIC GOROUTINES
│       ├── dispatcher.go             # Goroutine per topic message processing
│       └── server.go                 # Server lifecycle, signal handling, graceful drain
└── test/                             # INTEGRATION & E2E SCRIPTS
    ├── e2e_test.go                   # Automated end-to-end integration tests
    └── test_scenario.sh              # Bash test runner simulating subject flows
```

---

## 3. Core Technical Specifications

### 3.1 Custom Hashmap (`internal/hashmap`)
* **Strict Subject Rule**: Standard library maps are strictly forbidden for client metadata. Explicit collision resolution is mandatory.
* **Algorithm**: Separate Chaining with dynamic bucket array.
  * **Hash Function**: FNV-1a 64-bit hash modulo bucket count.
  * **Collision Resolution**: Linked list nodes (`Entry` struct with `Key`, `Value`, `Next`).
  * **Dynamic Resizing**: Automatically doubles bucket count when load factor exceeds 0.75.
  * **Thread Safety**: Protected with `sync.RWMutex`.
* **Metadata Schema**:
  ```go
  type ClientMetadata struct {
      ClientID  string // ^[a-zA-Z0-9_.-]{1,32}$
      Topic     string // ^[a-zA-Z0-9_.-]{1,32}$
      Offset    uint32 // Next offset client wants to consume
      Prefix    string // Optional key prefix filter
      IPCPath   string // Path to dedicated consumer FIFO
      IsActive  bool   // Whether consumer is actively connected
  }
  ```

### 3.2 Prefix Matching Trie (`internal/trie`)
* **Data Structure**: Prefix Trie where each node represents a character/byte.
* **Wildcard Optimization**: Empty prefix `""` consumers are stored in a root direct-dispatch slice (`wildcardConsumers`). They are added directly without traversing the Trie.
* **Prefix Matching**:
  * For message key `user.login`, lookups collect consumers registered under `""`, `"u"`, `"user"`, `"user."`, `"user.login"`.
  * Exact match: `user` matches key `user`.
  * Prefix match: `user` matches key `user.login`.
  * Non-match: `user` does NOT match key `admin`.

### 3.3 Message Formats & Wire Framing (`internal/protocol`)
* **Text Mode (Default)**:
  * Producer stdin: `key:body\n`
  * Consumer stdout: `key:body\n`
* **Raw Binary Mode (`--raw`)**:
  * Producer stdin: `[keysize:int32][key:bytes][valuesize:int32][value:bytes]`
  * Consumer stdout: `[offset:int32][keysize:int32][key:bytes][valuesize:int32][value:bytes]`
  * Little-endian binary encoding (`binary.LittleEndian`).
  * Back-to-back records without separator.
  * Partial record before EOF triggers exit code 1.
  * Maximum size: 1024 bytes (`key` + `body`, excluding metadata). Exceeding messages are rejected.

### 3.4 Offset & Acknowledgment Semantics
* Server maintains a zero-based incrementing 32-bit offset per topic.
* **Committed Offset Rule**: The stored offset is always the **next topic offset** the client wants to consume ($N + 1$).
* When a consumer receives offset $N$, it sends an ACK with $N + 1$.
* Returning consumers automatically resume from their stored offset unless an explicit `--offset <val>` is provided.

### 3.5 Exit Code Precedence
Strict priority hierarchy when multiple conditions apply:
1. **Code 1 (Highest Precedence)**: General errors, invalid CLI args, regex mismatch on client/topic name, connection failure, partial record at EOF.
2. **Code 3**: IPC communication errors, unexpected server disconnection.
3. **Code 2 (Lowest Precedence)**: Topic error (not found, already exists), client error (client not found for info, duplicate active subscriber name).
4. **Code 0**: Success, clean exit on SIGINT/SIGTERM or server shutdown.

---

## 4. Work Distribution: Balanced 3-Person Team

To guarantee strict parity in technical complexity, workload volume, and intellectual challenge, the project is divided into three distinct roles:

```
┌───────────────────────────┬───────────────────────────┬───────────────────────────┐
│     TEAMMATE 1 (A)        │     TEAMMATE 2 (B)        │     TEAMMATE 3 (C)        │
│  Data Structures & Engine │   IPC Subsystem & Server  │   Protocol, CLI & Testing │
├───────────────────────────┼───────────────────────────┼───────────────────────────┤
│ • Custom Hashmap (no std) │ • Pure Go FIFO (syscall)  │ • Wire framing (text/raw) │
│ • Prefix Matching Trie    │ • Server main loop & PID  │ • Little-endian codec     │
│ • Topic Message Storage   │ • Topic worker goroutines │ • Full Client CLI modes   │
│ • Unit Tests (Prefix/Map) │ • Shutdown & Sentinel     │ • Exit codes & Precedence │
│ • Benchmark suites        │ • Dedicated consumer pipe │ • Makefile, README, E2E   │
└───────────────────────────┴───────────────────────────┴───────────────────────────┘
```

### Role 1: Core Data Structures & Storage Specialist (Teammate A)
* **Technical Focus**: Algorithms, memory management, algorithmic correctness, and unit testing.
* **Responsibilities**:
  1. **Custom Hashmap (`internal/hashmap`)**:
     - Implement from scratch without `map`: buckets, nodes, hashing (FNV-1a), collision resolution (chaining), growth factor / rehashing.
     - Implement thread-safe read/write methods and iterator for client metadata.
  2. **Prefix Matching Structure (`internal/trie`)**:
     - Implement Trie/Radix Tree indexing consumers by prefix.
     - Implement direct-dispatch wildcard list for empty prefix `""`.
     - Implement prefix lookup matching `exact`, `partial`, and `no match`.
  3. **Topic Memory Buffer (`internal/storage`)**:
     - In-memory message store per topic with 32-bit offset generation.
     - Thread-safe append and offset-based retrieval.
  4. **Comprehensive Unit Testing**:
     - Exhaustive unit tests for Prefix Matching required by Chapter VI.11 (exact, partial, empty prefix, edge cases, large scale).
     - Full test coverage for hashmap collision handling.
* **Evaluation Deliverable**: Teammate A will defend the custom data structures and the algorithmic performance during peer evaluation.

---

### Role 2: IPC Subsystem & Server Concurrency Specialist (Teammate B)
* **Technical Focus**: Systems programming, OS-level IPC, goroutine orchestration, signal handling, and synchronization.
* **Responsibilities**:
  1. **Pure Go Named Pipe Engine (`internal/ipc`)**:
     - Implement `syscall.Mkfifo`, robust file descriptor handling, non-blocking / buffered opens, and pipe cleanup.
     - Create server main endpoint `/tmp/treenity.server.<PID>` and manage endpoint discovery.
     - Implement bidirectional request-response transport using temporary client reply pipes.
  2. **Dedicated Consumer Channels**:
     - Manage creation, writing, and teardown of dedicated consumer FIFOs `/tmp/treenity.server.<PID>.<client_id>`.
  3. **Server Daemon & Topic Goroutines (`internal/engine`, `cmd/server`)**:
     - Implement at least one dedicated worker goroutine per topic to handle inbound messages and consumer routing.
     - Implement server main dispatch loop handling management commands (`create`, `list`, `info`).
  4. **Graceful Shutdown & Sentinel Architecture**:
     - Capture `SIGINT` / `SIGTERM` signals.
     - Broadcast shutdown sentinel records across all consumer pipes to wake up blocked consumers and exit cleanly with code 0.
     - Enforce 5-second drain timeout before final cleanup.
* **Evaluation Deliverable**: Teammate B will defend the IPC choice, FIFO mechanics, bidirectional protocol, topic concurrency, and shutdown sequence.

---

### Role 3: Protocol, CLI Client & Integration Specialist (Teammate C)
* **Technical Focus**: Serialization, CLI interface, validation, error hierarchy, build automation, and end-to-end integration.
* **Responsibilities**:
  1. **Protocol & Wire Serialization (`internal/protocol`)**:
     - Implement text encoder/decoder (`key:body\n`).
     - Implement binary encoder/decoder (`[keysize:int32][key:bytes][valuesize:int32][value:bytes]`, little-endian, contiguous framing).
     - Implement EOF partial record detection (triggering exit code 1).
     - Implement request/response packet framing for IPC commands.
  2. **Client CLI Executable (`cmd/client`)**:
     - Implement subcommands: `create`, `list`, `produce`, `subscribe`, `info`.
     - Implement flags: `--raw`, `--prefix`, `--offset`.
     - Ensure strict stdout formatting (protocol only) and stderr logging (diagnostics/errors).
     - Implement consumer signal handling (`SIGINT`/`SIGTERM` sends disconnect, cleans up, exits 0).
  3. **Exit Code Precedence Engine**:
     - Enforce strict priority: `1 (General/Args)` > `3 (IPC)` > `2 (Topic/Client)`.
     - Implement input validation: regex `^[a-zA-Z0-9_.-]{1,32}$` for client IDs and topic names.
  4. **Build Automation, README & Integration Suite**:
     - Write root `Makefile` implementing `all`, `clean`, `re`, `test`.
     - Write `README.md` following exact 42 template requirements.
     - Write automated end-to-end test script (`test/test_scenario.sh`) verifying full pipeline against example scenarios.
* **Evaluation Deliverable**: Teammate C will defend the CLI argument parser, protocol byte ordering, error handling precedence, and integration test setup.

---

## 5. Git Workflow & 42 Evaluation Guidelines

Compliance with `git_guidelines.pdf` is strictly evaluated using `gitinette`. A single failure results in **0/100**.

### 5.1 Branching Strategy
* The `main` branch is strictly protected. Only minor non-functional changes (typo in docs) may be committed directly to `main`.
* Every feature, fix, or chore must be developed on a dedicated branch named according to its type:
  - `feat/<feature-name>` (e.g. `feat/custom-hashmap`, `feat/ipc-fifo-engine`, `feat/client-cli`)
  - `fix/<issue-name>` (e.g. `fix/sentinel-wakeup`, `fix/offset-acknowledgment`)
  - `chore/<task-name>` (e.g. `chore/makefile-setup`, `chore/readme-docs`)

### 5.2 Commit Message Rules (Conventional Commits)
Format: `<type>(<scope>): <short description in lowercase>`
* Valid types: `feat`, `fix`, `chore`, `test`, `refactor`, `docs`.
* Rules:
  - One logical change per commit.
  - Never mix refactoring with feature code.
  - Intermediate commits on feature branches are encouraged.
* Examples:
  - `feat(hashmap): implement separate chaining with fnv-1a hashing`
  - `feat(trie): add prefix search and wildcard empty prefix handling`
  - `feat(ipc): implement named pipe creation and unlink cleanup`
  - `feat(client): add subscribe subcommand with offset tracking`
  - `test(trie): add edge case tests for prefix matching`
  - `fix(protocol): enforce little-endian integer encoding for raw binary`

### 5.3 Merging Strategy (Strictly `--no-ff`)
* When a feature branch is ready:
  1. Rebase on latest main:
     ```bash
     git checkout feat/my-feature
     git rebase main
     ```
  2. Switch to main and merge with non-fast-forward:
     ```bash
     git checkout main
     git merge --no-ff feat/my-feature -m "feat: merge feat/my-feature"
     ```
* **FORBIDDEN**:
  - `git merge --squash` is strictly prohibited.
  - Merging `main` into a feature branch with a merge commit is strictly prohibited. Use `git rebase main`.

### 5.4 Local Verification with `gitinette`
Before submission, run `gitinette` locally to verify:
```bash
./gitinette
```
During evaluation, the peer evaluator will run:
```bash
./gitinette --evaluation
```
which validates branch naming, commit syntax, and performs a random commit message/diff consistency check.

---

## 6. Implementation Roadmap & Milestones

```
Phase 1: Project Setup & Contracts       (Days 1 - 2)
  ├── Git repo initialization & branch structure
  ├── Shared interfaces: internal/protocol/types.go
  └── Makefile skeleton (all, clean, re, test)

Phase 2: Component Implementation         (Days 2 - 4)
  ├── Teammate A: Hashmap, Trie, Topic Buffer & Unit Tests
  ├── Teammate B: Named Pipe IPC, Server Engine & Goroutines
  └── Teammate C: Binary/Text Codecs, CLI Subcommands & Flags

Phase 3: Subsystem Integration           (Days 4 - 6)
  ├── Connect Client CLI -> IPC -> Server Dispatcher
  ├── Connect Topic Dispatcher -> Trie -> Dedicated Consumer FIFOs
  └── Acknowledgment & Offset ($N+1$) persistence

Phase 4: Graceful Shutdown & Reliability (Days 6 - 7)
  ├── SIGINT/SIGTERM traps on Server & Client
  ├── Sentinel message delivery to wake up blocked consumers
  ├── 5-second drain timeout
  └── Exit code precedence enforcement (1 > 3 > 2)

Phase 5: Verification & Evaluation Prep   (Days 7 - 8)
  ├── Comprehensive prefix matching test suite (`go test -v ./...`)
  ├── End-to-end validation with ft_aquarium & ft_fish
  ├── README.md completion
  └── Local `gitinette` validation
```

---

## 7. Makefile Specification

The `Makefile` must be located at the root of the repository and support:

```makefile
# Variables
SERVER_BIN := server
CLIENT_BIN := client
GO         := go
SRC        := $(shell find . -type f -name '*.go')

.PHONY: all clean re test

all: $(SERVER_BIN) $(CLIENT_BIN)

$(SERVER_BIN): $(SRC)
	$(GO) build -o $(SERVER_BIN) ./cmd/server

$(CLIENT_BIN): $(SRC)
	$(GO) build -o $(CLIENT_BIN) ./cmd/client

clean:
	rm -f $(SERVER_BIN) $(CLIENT_BIN)

re: clean all

test:
	$(GO) test -v -race ./internal/...
```

---

## 8. README.md Requirements Checklist

The `README.md` must strictly include:
1. **Mandatory First Line** (Italicized):
   *`This project has been created as part of the 42 curriculum by <login1>, <login2>, <login3>.*`
2. **Description**: Clear presentation of the project, architecture, and goals.
3. **Instructions**: How to compile (`make`), run tests (`make test`), start the server, and execute client subcommands.
4. **Resources**: References and explicit disclosure of AI tool usage.
5. **Architecture**: Detailed explanation of the dedicated goroutine per topic and mutex synchronization.
6. **IPC Choice**: Complete justification of Named Pipes (FIFOs) and explanation of bidirectional communication via reply FIFOs.
7. **Data Structures**: Detailed description of the custom hashmap (collision resolution) and the Prefix Trie.
8. **Testing**: Guide on how unit tests are organized and executed.

---

## 9. Evaluation & "Recode" Defense Preparedness

During evaluation, evaluators may ask each team member to perform a brief live code modification ("Recode" requirement):
* **Teammate A**: Be prepared to modify the custom hashmap (e.g. add a `Count()` method or change the bucket size) or adjust the Trie search logic.
* **Teammate B**: Be prepared to modify the FIFO directory path, change the shutdown timeout, or add a debug logging hook to the topic worker goroutines.
* **Teammate C**: Be prepared to add a new CLI flag (e.g. `--verbose`), adjust the JSON output format of `info`, or add a custom exit code check.
