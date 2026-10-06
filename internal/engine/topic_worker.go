package engine

import (
	"Tree_nity/internal/ipc"
	"Tree_nity/internal/topic"
	"Tree_nity/internal/trie"
	"context"
	"fmt"
	"sync"
)

type InboundMessage struct {
	Key		string
	Payload []byte
	AckChan chan<- uint32
}

type TopicWorker struct {
	TopicName	string
	inboundChan	chan InboundMessage
	store		topic.TopicStore
	matcher		trie.Matcher
	consumers	map[string]*ipc.ConsumerChannel
	mu			sync.RWMutex
	ctx			context.Context
	cancel		context.CancelFunc
	wg			sync.WaitGroup
}

func NewTopicWorker(topicName string, store topic.TopicStore, matcher trie.Matcher) *TopicWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &TopicWorker{
		TopicName: topicName,
		inboundChan: make(chan InboundMessage, 1024),
		store:		 store,
		matcher: matcher,
		consumers: make(map[string]*ipc.ConsumerChannel),
		ctx: ctx,
		cancel: cancel,
	}
}

func (tw *TopicWorker) processMessage(msg InboundMessage) {
	offset := tw.store.Append(tw.TopicName, msg.Payload)
	if msg.AckChan != nil {
		select {
		case msg.AckChan <- offset:
		default:
		}
	}

	matchingClientIDs := tw.matcher.Match(msg.Key)

	tw.mu.RLock()
	defer tw.mu.RUnlock()

	for _, clientID := range matchingClientIDs {
		if ch, exists := tw.consumers[clientID]; exists {
			_ = ch.Push(msg.Payload)
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
	tw.consumers[clientID] = ch
	tw.matcher.Add(prefix, clientID)
	tw.mu.Unlock()

	historical := tw.store.ReadFrom(tw.TopicName, startOffset)
	for _, m := range historical {
		if err := ch.Push(m.Payload); err != nil {
			return fmt.Errorf("replay consumer %q failed: %w", clientID, err)
		}
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
