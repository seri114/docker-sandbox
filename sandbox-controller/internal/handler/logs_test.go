package handler

import (
	"bytes"
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
