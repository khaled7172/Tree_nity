package ipc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
)

func ServerEndpoint(pid int) string {
	return fmt.Sprintf("/tmp/operator.server.%d", pid)
}

func ConsumerEndpoint(serverPID int, clientID string) string {
	return fmt.Sprintf("/tmp/operator.server.%d.%s", serverPID, clientID)
}

func ClientReplyEndpoint(clientPID int, seq uint64) string {
	return fmt.Sprintf("/tmp/operator.client.%d.%d", clientPID, seq)
}

type ConsumerChannel struct {
	ClientID string
	Path	 string
	file	 *os.File
	mu		 sync.Mutex
	isClosed bool
}

func NewConsumerChannel(serverPID int, clientID string) (*ConsumerChannel, error) {
	path := ConsumerEndpoint(serverPID, clientID)

	if err := MakeFifo(path); err != nil {
		return nil, fmt.Errorf("create consumer fifo %q: %w", path, err)
	}

	file, err := os.OpenFile(path, os.O_RDWR, 0660)
	if err != nil {
		_ = RemoveFifo(path)
		return nil, fmt.Errorf("open consumer fifo: %q: %w", path, err)
	}

	return &ConsumerChannel{
		ClientID: clientID,
		Path: path,
		file: file,
	}, nil
}

func (c *ConsumerChannel) Push(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosed {
		return errors.New("consumer channel is closed")
	}

	_, err := c.file.Write(data)
	if err != nil {
		if errors.Is(err, syscall.EPIPE) {
			return fmt.Errorf("consumer %q disconnected (broken pipe): %w", c.ClientID, err)
		}
		return fmt.Errorf("write to consumer %q: %w", c.ClientID, err)
	}

	return nil
}

func (c *ConsumerChannel) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosed {
		return nil
	}
	c.isClosed = true

	var closeErr error
	if c.file != nil {
		closeErr = c.file.Close()
	}

	removeErr := RemoveFifo(c.Path)
	if closeErr != nil {
		return closeErr
	}
	return removeErr
}

func WriteReply(ctx context.Context, replyPath string, response []byte) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		fd, err := syscall.Open(replyPath, syscall.O_WRONLY|syscall.O_NONBLOCK, 0660)
		if err == nil {
			_ = syscall.SetNonblock(fd, false)
			file := os.NewFile(uintptr(fd), replyPath)
			defer file.Close()

			if _, err := file.Write(response); err != nil {
				return fmt.Errorf("write response to %q: %w", replyPath, err)
			}
			return nil
		}

		if !errors.Is(err, syscall.ENXIO) {
			return fmt.Errorf("open reply fifo %q: %w", replyPath, err)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for client reply reader %q: %w", replyPath, ctx.Err())
		case <-ticker.C:
		}
	}
}

func WriteReplyWithTimeout(replyPath string, response []byte, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return WriteReply(ctx, replyPath, response)
}
