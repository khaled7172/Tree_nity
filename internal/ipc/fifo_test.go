package ipc

import (
	"bytes"
	"path/filepath"
	"sync"
	"testing"
)

func TestMakeAndRemoveFIFO(t *testing.T) {
	tmpDir := t.TempDir()
	fifoPath := filepath.Join(tmpDir, "test.fifo")

	if err := MakeFifo(fifoPath); err != nil {
		t.Fatalf("MakeFIFO failed: %v", err)
	}

	if !IsFifo(fifoPath) {
		t.Fatalf("expected %s to be recognized as FIFO", fifoPath)
	}

	if err := MakeFifo(fifoPath); err != nil {
		t.Fatalf("MakeFIFO over existing path failed: %v", err)
	}

	if err := RemoveFifo(fifoPath); err != nil {
		t.Fatalf("RemoveFIFO failed: %v", err)
	}

	if IsFifo(fifoPath) {
		t.Fatalf("expected %s to be removed", fifoPath)
	}

	if err := RemoveFifo(fifoPath); err != nil {
		t.Fatalf("RemoveFIFO on non-existent path returned error: %v", err)
	}
}

func TestFIFOReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	fifoPath := filepath.Join(tmpDir, "rw.fifo")

	if err := MakeFifo(fifoPath); err != nil {
		t.Fatalf("MakeFIFO failed: %v", err)
	}
	defer RemoveFifo(fifoPath)

	testMessage := []byte("hello treenity ipc")
	readBuf := make([]byte, len(testMessage))

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		r, err := OpenReader(fifoPath)
		if err != nil {
			t.Errorf("OpenReader failed: %v", err)
			return
		}
		defer r.Close()

		_, err = r.Read(readBuf)
		if err != nil {
			t.Errorf("Reader read failed: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		w, err := OpenWriter(fifoPath)
		if err != nil {
			t.Errorf("OpenWriter failed: %v", err)
			return
		}
		defer w.Close()

		_, err = w.Write(testMessage)
		if err != nil {
			t.Errorf("Writer write failed: %v", err)
		}
	}()

	wg.Wait()

	if !bytes.Equal(readBuf, testMessage) {
		t.Fatalf("expected message %q, got %q", testMessage, readBuf)
	}
}

func TestOpenServerFIFORDWR(t *testing.T) {
	tmpDir := t.TempDir()
	fifoPath := filepath.Join(tmpDir, "server.fifo")

	if err := MakeFifo(fifoPath); err != nil {
		t.Fatalf("MakeFIFO failed: %v", err)
	}
	defer RemoveFifo(fifoPath)

	serverFile, err := OpenServerFifo(fifoPath)
	if err != nil {
		t.Fatalf("OpenServerFIFO failed: %v", err)
	}
	defer serverFile.Close()

	msg := []byte("self write check")
	if _, err := serverFile.Write(msg); err != nil {
		t.Fatalf("failed to write to O_RDWR FIFO: %v", err)
	}

	buf := make([]byte, len(msg))
	if _, err := serverFile.Read(buf); err != nil {
		t.Fatalf("failed to read from O_RDWR FIFO: %v", err)
	}

	if !bytes.Equal(buf, msg) {
		t.Fatalf("expected %q, got %q", msg, buf)
	}
}
