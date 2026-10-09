package engine

import (
	"Tree_nity/internal/ipc"
	"time"
)

const ShutdownSentinel = "__TREENITY_SHUTDOWN__\n"

func (s *Server) Shutdown(timeout time.Duration) error {
	s.mu.Lock()
	sentinelBytes := []byte(ShutdownSentinel)
	for _, ch := range s.consumers {
		_ = ch.Push(sentinelBytes)
	}
	s.mu.Unlock()

	drainTimer := time.NewTimer(timeout)
	defer drainTimer.Stop()

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	drainLoop:
		for {
			s.mu.RLock()
			activeCount := len(s.consumers)
			s.mu.RUnlock()

			if activeCount == 0 {
				break drainLoop
			}

			select {
			case <-drainTimer.C:
				break drainLoop
			case <-ticker.C:
			}
		}

		s.cancel()
		if s.file != nil {
			_, _ = s.file.Write([]byte("{}\n"))
			_ = s.file.Close()
		}
		s.wg.Wait()

		s.mu.Lock()
		defer s.mu.Unlock()

		for _, worker := range s.topicWorkers {
			worker.Stop()
		}

		for _, ch := range s.consumers {
			_ = ch.Close()
		}
		s.consumers = make(map[string]*ipc.ConsumerChannel)

		return ipc.RemoveFifo(s.Endpoint)
}
