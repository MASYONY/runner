package utils

import (
	"io"
	"log"
	"net"
	"os"
	"sync"
)

var (
	InfoLogger  = log.New(os.Stdout, "INFO: ", log.LstdFlags)
	ErrorLogger = log.New(os.Stderr, "ERROR: ", log.LstdFlags)
)

type SocketLogWriter struct {
	conn net.Conn
	mu   sync.Mutex
}

func NewSocketLogWriter(socketPath string) (*SocketLogWriter, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}
	return &SocketLogWriter{conn: conn}, nil
}

func (w *SocketLogWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.Write(p)
}

func SetOutput(w io.Writer) {
	InfoLogger.SetOutput(w)
	ErrorLogger.SetOutput(w)
}

func InitSocketLogging() {
	socketPath := os.Getenv("RUNNER_LOG_SOCKET")
	if socketPath == "" {
		return
	}
	writer, err := NewSocketLogWriter(socketPath)
	if err == nil {
		mw := io.MultiWriter(os.Stdout, writer)
		InfoLogger.SetOutput(mw)
		ErrorLogger.SetOutput(mw)
	} else {
		// Fallback: nur stdout/stderr
		InfoLogger.SetOutput(os.Stdout)
		ErrorLogger.SetOutput(os.Stderr)
	}
}
