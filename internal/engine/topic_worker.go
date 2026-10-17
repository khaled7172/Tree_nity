package engine

import (
	"Tree_nity/internal/hashmap"
	"Tree_nity/internal/ipc"
	"Tree_nity/internal/topic"
	"Tree_nity/internal/trie"
	"context"
	"fmt"
	"strings"
	"sync"
)

type InboundMessage struct {
	Key     string
	Payload []byte
}

type TopicWorker struct {
	TopicName   string
	inboundChan chan InboundMessage
	store       topic.TopicStore
	matcher     trie.Matcher
	clientMap   hashmap.ClientMap
	consumers   map[string]*ipc.ConsumerChannel
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func NewTopicWorker(topicName string, store topic.TopicStore, matcher trie.Matcher, clientMap hashmap.ClientMap) *TopicWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &TopicWorker{
		TopicName:   topicName,
		inboundChan: make(chan InboundMessage, 1024),
		store:       store,
		matcher:     matcher,
		clientMap:   clientMap,
		consumers:   make(map[string]*ipc.ConsumerChannel),
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (tw *TopicWorker) processMessage(msg InboundMessage) {
	offset := tw.store.Append(tw.TopicName, msg.Key, msg.Payload)
	matchingClientIDs := tw.matcher.Match(msg.Key)

	tw.mu.Lock()
	defer tw.mu.Unlock()

	for _, clientID := range matchingClientIDs {
		if ch, exists := tw.consumers[clientID]; exists {
			_ = ch.Push(msg.Payload)
		}
	}

	if tw.clientMap != nil {
		for clientID := range tw.consumers {
			_ = tw.clientMap.UpdateOffset(clientID, offset+1)
		}
	}
}

func (tw *TopicWorker) Start() {
	tw.wg.Add(1)
	go func() {
		defer tw.wg.Done()
		for {
			select {
			case <-tw.ctx.Done():
				return
			case msg, ok := <-tw.inboundChan:
				if !ok {
					return
				}
				tw.processMessage(msg)
			}
		}
	}()
}

func (tw *TopicWorker) Publish(ctx context.Context, msg InboundMessage) error {
	select {
	case <-tw.ctx.Done():
		return fmt.Errorf("topic worker for %q is stopped", tw.TopicName)
	case <-ctx.Done():
		return ctx.Err()
	case tw.inboundChan <- msg:
		return nil
	}
}

func (tw *TopicWorker) Subscribe(clientID string, prefix string, startOffset uint32, ch *ipc.ConsumerChannel) error {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	historical := tw.store.ReadFrom(tw.TopicName, startOffset)
	for _, m := range historical {
		if prefix != "" && !strings.HasPrefix(m.Key, prefix) {
			continue
		}
		if err := ch.Push(m.Payload); err != nil {
			return fmt.Errorf("replay consumer %q failed: %w", clientID, err)
		}
	}

	tw.consumers[clientID] = ch
	tw.matcher.Add(prefix, clientID)

	if tw.clientMap != nil {
		_ = tw.clientMap.UpdateOffset(clientID, tw.store.LatestOffset(tw.TopicName))
	}

	return nil
}

func (tw *TopicWorker) Unsubscribe(clientID string, prefix string) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	tw.matcher.Remove(prefix, clientID)
	delete(tw.consumers, clientID)
}

func (tw *TopicWorker) Stop() {
	tw.cancel()
	tw.wg.Wait()
}
