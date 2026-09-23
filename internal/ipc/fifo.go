package ipc

import (
	"os"
	"fmt"
	"syscall"
)

const DefaultFIFOMode uint32 = 0660

func MakeFifo(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove existing fifo %q: %w", path, err)
	}

	if err := syscall.Mkfifo(path, DefaultFIFOMode); err != nil {
		return fmt.Errorf("mkdifof %q: %w", path, err)
	}
	return nil
}

func RemoveFifo(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove fifo %q: %w", path, err)
	}
	return nil
}

func IsFifo(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeNamedPipe) != 0
}

func OpenServerFifo(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open server fifo %q (O_RDWR): %w", path, err)
	}
	return file, nil
}

func OpenReader(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("open reader fifo %q: %w", path, err)
	}
	return file, nil
}

func OpenWriter(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("open writer fifo %q: %w", path, err)
	}
	return file, nil
}
