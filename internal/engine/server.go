package engine

import (
	"Tree_nity/internal/hashmap"
	"Tree_nity/internal/ipc"
	"Tree_nity/internal/protocol"
	"Tree_nity/internal/topic"
	"Tree_nity/internal/trie"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type Server struct {
	PID				int
	Endpoint		string
	file			*os.File
	topicStore		topic.TopicStore
	clientMap		hashmap.ClientMap
	topicWorkers	map[string]*TopicWorker
	consumers		map[string]*ipc.ConsumerChannel
	mu				sync.RWMutex
	ctx				context.Context
	cancel			context.CancelFunc
	wg				sync.WaitGroup
}

func NewServer(pid int, store topic.TopicStore, clientMap hashmap.ClientMap) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		PID: pid,
		Endpoint: ipc.ServerEndpoint(pid),
		topicStore: store,
		clientMap: clientMap,
		consumers: make(map[string]*ipc.ConsumerChannel),
		topicWorkers: make(map[string]*TopicWorker),
		ctx: ctx,
		cancel: cancel,
	}
}

func (s *Server) handleRequest(req protocol.Request) {
	switch req.Type {
	case protocol.CmdCreate:
		s.handleCreate(req)
	case protocol.CmdList:
		s.handleList(req)
	case protocol.CmdInfo:
		s.handleInfo(req)
	case protocol.CmdSubscribe:
		s.handleSubscribe(req)
	case protocol.CmdProduce:
		s.handleProduce(req)
	case protocol.CmdDrainAck:
		s.handleDrainAck(req)
	}
}

func (s *Server) dispatchLoop() {
	defer s.wg.Done()

	scanner := bufio.NewScanner(s.file)

	buf := make([]byte, 1024*64)
	scanner.Buffer(buf, len(buf))

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req protocol.Request
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		s.handleRequest(req)
	}
}

func (s *Server) Start() error {
	if err := ipc.MakeFifo(s.Endpoint); err != nil {
		return fmt.Errorf("create server fifo: %w", err)
	}

	file, err := ipc.OpenServerFifo(s.Endpoint)
	if err != nil {
		_ = ipc.RemoveFifo(s.Endpoint)
		return fmt.Errorf("open server fifo: %w", err)
	}
	s.file = file

	fmt.Println(s.Endpoint)

	s.wg.Add(1)
	go s.dispatchLoop()

	return nil
}

func (s *Server) Stop() error {
	s.cancel()

	if s.file != nil {
		_ = s.file.Close()
	}

	s.wg.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, worker := range s.topicWorkers {
		worker.Stop()
	}

	for _, ch := range s.consumers {
		ch.Close()
	}

	return ipc.RemoveFifo(s.Endpoint)
}

func (s *Server) handleCreate(req protocol.Request) {
	if !protocol.ValidateIdentifier(req.Topic) {
		s.sendReply(req.ReplyFIFO, protocol.Response{
			Success: false,
			ExitCode: protocol.ExitGeneral,
			Message: "invalid topic name",
		})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.topicWorkers[req.Topic]; exists {
		s.sendReply(req.ReplyFIFO, protocol.Response{
			Success: false,
			ExitCode: protocol.ExitTopic,
			Message: "topic already exists",
		})
		return
	}

	worker := NewTopicWorker(req.Topic, s.topicStore, trie.New())
	worker.Start()
	s.topicWorkers[req.Topic] = worker

	s.sendReply(req.ReplyFIFO, protocol.Response{
		Success: true,
		ExitCode: protocol.ExitSuccess,
		Message: "topic created",
	})
}

func (s *Server) handleList(req protocol.Request) {
	s.mu.RLock()
	topics := make([]string, 0, len(s.topicWorkers))
	for name := range s.topicWorkers {
		topics = append(topics, name)
	}
	s.mu.RUnlock()

	s.sendReply(req.ReplyFIFO, protocol.Response{
		Success: true,
		ExitCode: protocol.ExitSuccess,
		Topics: topics,
	})
}

func (s *Server) handleInfo(req protocol.Request) {
	if !protocol.ValidateIdentifier(req.ClientID) {
		s.sendReply(req.ReplyFIFO, protocol.Response{
			Success: false,
			ExitCode: protocol.ExitGeneral,
			Message: "invalid client name",
		})
		return
	}

	meta, found := s.clientMap.Get(req.ClientID)
	if !found {
		s.sendReply(req.ReplyFIFO, protocol.Response{
			Success: false,
			ExitCode: protocol.ExitTopic,
			Message: "client not found",
		})
		return
	}

	s.sendReply(req.ReplyFIFO, protocol.Response{
		Success: true,
		ExitCode: protocol.ExitSuccess,
		Metadata: &protocol.ClientInfo{
			Client: meta.ClientID,
			Topic: meta.Topic,
			Offset: meta.Offset,
			Prefix: meta.Prefix,
			IPC: meta.IPCPath,
		},
	})
}

func (s *Server) handleSubscribe(req protocol.Request) {
	if !protocol.ValidateIdentifier(req.Topic) || !protocol.ValidateIdentifier(req.ClientID) {
		s.sendReply(req.ReplyFIFO, protocol.Response{
			Success: false,
			ExitCode: protocol.ExitGeneral,
			Message: "invalid topic or client name",
		})
		return
	}

	s.mu.Lock()
	worker, topicExists := s.topicWorkers[req.Topic]
	if !topicExists {
		s.mu.Unlock()
		s.sendReply(req.ReplyFIFO, protocol.Response{
			Success: false,
			ExitCode: protocol.ExitTopic,
			Message: "topic does not exist",
		})
		return
	}

	if existing, found := s.clientMap.Get(req.ClientID); found && existing.IsActive {
		s.mu.Unlock()
		s.sendReply(req.ReplyFIFO, protocol.Response{
			Success: false,
			ExitCode: protocol.ExitTopic,
			Message: "duplicate client name",
		})
		return
	}

	var startOffset uint32
	if req.Offset != nil {
		startOffset = *req.Offset
	} else if existing, found := s.clientMap.Get(req.ClientID); found {
		startOffset = existing.Offset
	} else {
		startOffset = 0
	}

	ch, err := ipc.NewConsumerChannel(s.PID, req.ClientID)
	if err != nil {
		s.mu.Unlock()
		s.sendReply(req.ReplyFIFO, protocol.Response{
			Success: false,
			ExitCode: protocol.ExitIPC,
			Message: "failed to create consumer channel",
		})
		return
	}

	s.consumers[req.ClientID] = ch
	_ = s.clientMap.Put(hashmap.ClientMetadata{
		ClientID: req.ClientID,
		Topic: req.Topic,
		Offset: startOffset,
		Prefix: req.Prefix,
		IPCPath: ch.Path,
		IsActive: true,
	})

	_ = worker.Subscribe(req.ClientID, req.Prefix, startOffset, ch)
	s.mu.Unlock()

	s.sendReply(req.ReplyFIFO, protocol.Response{
		Success: true,
		ExitCode: protocol.ExitSuccess,
		Message: fmt.Sprintf("subscribed to %s", req.Topic),
	})
}

func (s *Server) handleProduce(req protocol.Request) {
	s.mu.RLock()
	_, exists := s.topicWorkers[req.Topic]
	s.mu.RUnlock()

	if !exists {
		s.sendReply(req.ReplyFIFO, protocol.Response{
			Success: false,
			ExitCode: protocol.ExitTopic,
			Message: "topic does not exist",
		})
		return
	}

	s.sendReply(req.ReplyFIFO, protocol.Response{
		Success: true,
		ExitCode: protocol.ExitSuccess,
		Message: "topic exists",
	})
}

func (s *Server) handleDrainAck(req protocol.Request) {
	if req.ClientID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if req.Offset != nil {
		_ = s.clientMap.UpdateOffset(req.ClientID, *req.Offset)
	}

	if meta, found := s.clientMap.Get(req.ClientID); found {
		meta.IsActive = false
		if req.Offset != nil {
			meta.Offset = *req.Offset
		}
		_ = s.clientMap.Put(meta)
	}

	if ch, exists := s.consumers[req.ClientID]; exists {
		_ = ch.Close()
		delete(s.consumers, req.ClientID)
	}

	if req.ReplyFIFO != "" {
		s.sendReply(req.ReplyFIFO, protocol.Response{
			Success: true,
			ExitCode: protocol.ExitSuccess,
			Message: "acknowledged",
		})
	}
}

func (s *Server) sendReply(replyFIFO string, resp protocol.Response) {
	if replyFIFO == "" {
		return
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	data = append(data, '\n')
	_ = ipc.WriteReplyWithTimeout(replyFIFO, data, 2*time.Second)
}
