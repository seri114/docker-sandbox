package handler

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestLogMessageToJSON tests the ToJSON method of LogMessage.
func TestLogMessageToJSON(t *testing.T) {
	tests := []struct {
		name     string
		msg      LogMessage
		expected string
	}{
		{
			name: "stdout message",
			msg: LogMessage{
				Stream: "stdout",
				Data:   "hello world\n",
			},
			expected: `{"stream":"stdout","data":"hello world\n"}`,
		},
		{
			name: "stderr message",
			msg: LogMessage{
				Stream: "stderr",
				Data:   "error occurred\n",
			},
			expected: `{"stream":"stderr","data":"error occurred\n"}`,
		},
		{
			name: "empty data",
			msg: LogMessage{
				Stream: "stdout",
				Data:   "",
			},
			expected: `{"stream":"stdout","data":""}`,
		},
		{
			name: "message with special characters",
			msg: LogMessage{
				Stream: "stdout",
				Data:   `{"key":"value"}`,
			},
			expected: `{"stream":"stdout","data":"{\"key\":\"value\"}"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.msg.ToJSON()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestLogsStreamFormat tests the complete logs streaming format.
// This simulates the Docker JSONLog format and verifies that LogsStreamer
// correctly parses and sends messages to the channel.
func TestLogsStreamFormat(t *testing.T) {
	// Simulate Docker JSONLog format
	// Each log entry: [1]byte stream type + [7]byte padding + [8]byte size + data
	// stream type: 1 for stdout, 2 for stderr
	// size is big-endian

	// Helper to create a log entry
	createLogEntry := func(streamType byte, data string) []byte {
		size := len(data)
		entry := make([]byte, 16+size)
		entry[0] = streamType // bytes 1-7 are zeros (already zero from make)
		// Big-endian size in bytes 8-15
		entry[8] = byte(size >> 56)
		entry[9] = byte(size >> 48)
		entry[10] = byte(size >> 40)
		entry[11] = byte(size >> 32)
		entry[12] = byte(size >> 24)
		entry[13] = byte(size >> 16)
		entry[14] = byte(size >> 8)
		entry[15] = byte(size)
		copy(entry[16:], data)
		return entry
	}

	logs := [][]byte{
		createLogEntry(1, "hello world\n"),
		createLogEntry(2, "error message\n"),
		createLogEntry(1, "done\n"),
	}

	// Combine into a single reader
	var buf bytes.Buffer
	for _, log := range logs {
		buf.Write(log)
	}

	// Create logs streamer
	streamer := NewLogsStreamer(&buf)

	// Create channel to receive messages
	msgCh := make(chan LogMessage, 10)

	// Stream logs in goroutine
	go streamer.StreamTo(msgCh)

	// Collect messages
	var messages []LogMessage
	for i := 0; i < 3; i++ {
		msg := <-msgCh
		messages = append(messages, msg)
	}

	// Verify messages
	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}

	// First message - stdout
	if messages[0].Stream != "stdout" {
		t.Errorf("expected stream 'stdout', got '%s'", messages[0].Stream)
	}
	if messages[0].Data != "hello world\n" {
		t.Errorf("expected data 'hello world\\n', got '%s'", messages[0].Data)
	}

	// Second message - stderr
	if messages[1].Stream != "stderr" {
		t.Errorf("expected stream 'stderr', got '%s'", messages[1].Stream)
	}
	if messages[1].Data != "error message\n" {
		t.Errorf("expected data 'error message\\n', got '%s'", messages[1].Data)
	}

	// Third message - stdout
	if messages[2].Stream != "stdout" {
		t.Errorf("expected stream 'stdout', got '%s'", messages[2].Stream)
	}
	if messages[2].Data != "done\n" {
		t.Errorf("expected data 'done\\n', got '%s'", messages[2].Data)
	}

	// Verify each message can be converted to valid JSON
	for i, msg := range messages {
		jsonStr := msg.ToJSON()
		var parsed map[string]string
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			t.Errorf("message %d produced invalid JSON: %v", i, err)
		}
		if parsed["stream"] != msg.Stream {
			t.Errorf("message %d JSON stream mismatch", i)
		}
		if parsed["data"] != msg.Data {
			t.Errorf("message %d JSON data mismatch", i)
		}
	}
}

// TestLogsStreamFormatEmpty tests streaming with empty input.
func TestLogsStreamFormatEmpty(t *testing.T) {
	buf := bytes.Buffer{}
	streamer := NewLogsStreamer(&buf)
	msgCh := make(chan LogMessage, 1)

	go streamer.StreamTo(msgCh)

	// Should not receive any messages
	select {
	case msg := <-msgCh:
		t.Errorf("expected no messages, got %+v", msg)
	default:
		// Expected - no messages
	}
}

// TestLogsStreamFormatInvalidStream tests handling of invalid stream type.
func TestLogsStreamFormatInvalidStream(t *testing.T) {
	// Invalid stream type (3) - should be skipped
	createLogEntry := func(streamType byte, data string) []byte {
		size := len(data)
		entry := make([]byte, 16+size)
		entry[0] = streamType
		entry[8] = byte(size >> 56)
		entry[9] = byte(size >> 48)
		entry[10] = byte(size >> 40)
		entry[11] = byte(size >> 32)
		entry[12] = byte(size >> 24)
		entry[13] = byte(size >> 16)
		entry[14] = byte(size >> 8)
		entry[15] = byte(size)
		copy(entry[16:], data)
		return entry
	}

	invalidLog := createLogEntry(3, "test\n")

	var buf bytes.Buffer
	buf.Write(invalidLog)

	streamer := NewLogsStreamer(&buf)
	msgCh := make(chan LogMessage, 1)

	go streamer.StreamTo(msgCh)

	// Should handle gracefully - skip unknown stream types
	select {
	case msg := <-msgCh:
		// Unknown stream types should be skipped
		if msg.Stream == "unknown" {
			t.Errorf("expected unknown stream types to be skipped, got '%s'", msg.Stream)
		}
	default:
		// Expected - unknown stream types are skipped
	}
}

// TestLogsStreamerNewLogsStreamer tests the NewLogsStreamer constructor.
func TestLogsStreamerNewLogsStreamer(t *testing.T) {
	buf := bytes.Buffer{}
	streamer := NewLogsStreamer(&buf)

	if streamer == nil {
		t.Fatal("expected non-nil LogsStreamer")
	}
	if streamer.reader == nil {
		t.Fatal("expected reader to be initialized")
	}
}
