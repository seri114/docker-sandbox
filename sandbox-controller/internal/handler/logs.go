package handler

import (
	"encoding/json"
	"fmt"
	"io"
)

// LogMessage represents a single log message from a container.
// It is designed to be sent via Server-Sent Events (SSE) to clients.
type LogMessage struct {
	Stream string `json:"stream"` // Stream type: "stdout" or "stderr"
	Data   string `json:"data"`   // Log data content
}

// ToJSON converts the LogMessage to a JSON string.
// This is used for SSE event data formatting.
func (m *LogMessage) ToJSON() string {
	bytes, err := json.Marshal(m)
	if err != nil {
		// Fallback to manually constructed JSON on marshal error
		return fmt.Sprintf(`{"stream":"%s","data":"%s"}`, m.Stream, escapeJSONString(m.Data))
	}
	return string(bytes)
}

// escapeJSONString provides basic JSON escaping for string values.
func escapeJSONString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// LogsStreamer handles streaming of Docker container logs.
// It reads Docker JSONLog format and converts it to LogMessages.
type LogsStreamer struct {
	reader io.Reader
}

// NewLogsStreamer creates a new LogsStreamer with the given reader.
// The reader should provide Docker JSONLog formatted data.
func NewLogsStreamer(reader io.Reader) *LogsStreamer {
	return &LogsStreamer{
		reader: reader,
	}
}

// StreamTo reads logs from the reader and sends them to the channel.
// Each log entry is parsed from Docker JSONLog format (header + data)
// and converted to a LogMessage.
//
// Docker JSONLog format per line:
// - [8]byte stream type (1=stdout, 2=stderr)
// - [8]byte size of the data (little endian)
// - [size]byte data content
//
// The method closes the channel when done.
func (s *LogsStreamer) StreamTo(ch chan<- LogMessage) {
	defer close(ch)

	header := make([]byte, 16)
	buf := make([]byte, 4096)

	for {
		// Read header: stream type (1 byte) + padding (7 bytes) + size (8 bytes)
		// Docker uses: first byte is stream type, next 7 are zeros, then 8 bytes for size
		_, err := io.ReadFull(s.reader, header)
		if err != nil {
			return
		}

		// First byte is stream type (1=stdout, 2=stderr)
		streamType := header[0]

		// Bytes 8-15 contain the size (big endian in Docker's format)
		size := uint64(header[8])<<56 | uint64(header[9])<<48 | uint64(header[10])<<40 |
			uint64(header[11])<<32 | uint64(header[12])<<24 | uint64(header[13])<<16 |
			uint64(header[14])<<8 | uint64(header[15])

		// Read data
		var data []byte
		if size > 0 {
			if size <= uint64(len(buf)) {
				data = buf[:size]
			} else {
				data = make([]byte, size)
			}
			_, err := io.ReadFull(s.reader, data)
			if err != nil {
				return
			}
		}

		// Convert stream type to string
		stream := "unknown"
		switch streamType {
		case 1:
			stream = "stdout"
		case 2:
			stream = "stderr"
		}

		// Send message to channel (skip unknown stream types)
		if stream != "unknown" {
			ch <- LogMessage{
				Stream: stream,
				Data:   string(data),
			}
		}
	}
}
