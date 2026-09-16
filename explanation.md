# Treenity Project Explanation

## Overview
**Treenity** is a Message Queue System built as part of the 42 curriculum. It facilitates asynchronous communication between applications using an inter-process communication (IPC) transport layer. Think of it like a mini-Kafka where producers send messages, and consumers read them at their own pace.

## Core Components
1. **Server (`server`)**: 
   - Acts as the central broker.
   - Accepts multiple client connections using your chosen IPC mechanism.
   - Manages **topics** (named message channels) and stores messages in memory.
   - Routes messages to **consumers** based on their subscriptions, offsets, and prefix filters.
   - Handles graceful shutdowns when receiving SIGINT/SIGTERM.

2. **Client (`client`)**:
   - The executable that interfaces with the server. It can run in three modes:
   - **Management Mode**: Create new topics, list available topics, query client metadata (`info`).
   - **Producer Mode**: Sends messages (`key:body`) to a specific topic. Supports both text and raw binary formats.
   - **Consumer Mode**: Subscribes to a topic. Can filter incoming messages by key prefixes and resume reading from specific offsets (to replay old messages or skip ahead).

## Key Technical Requirements
- **IPC Mechanism**: You must choose and justify one of: System V Message Queues, POSIX Message Queues, or Named Pipes (FIFOs).
- **Concurrency**: Must handle concurrent access safely. Requires at least one dedicated thread/goroutine per topic for processing messages.
- **Data Structures**: 
  - **Client Indexing**: You *must* manually implement a hashmap with collision resolution (chaining or open addressing) for client metadata. Standard library maps are not allowed for this specific part.
  - **Prefix Matching**: Must implement an efficient structure (e.g., AVL tree, trie) to match consumers to message prefixes.
- **Language**: C++ (C++17+) or Go. (We have chosen Go 1.27 idiomatic).
- **Makefile**: Standard `all`, `clean`, `re`, `test` rules are required.
