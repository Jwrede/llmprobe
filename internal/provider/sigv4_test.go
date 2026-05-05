package provider

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"net/http"
	"strings"
	"testing"
)

func TestSignV4AddsHeaders(t *testing.T) {
	req, _ := http.NewRequest("POST", "https://bedrock-runtime.us-east-1.amazonaws.com/model/test/converse-stream", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")

	signV4(req, []byte("{}"), "us-east-1", "bedrock", "AKID", "SECRET")

	auth := req.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256") {
		t.Errorf("Authorization header missing or wrong: %q", auth)
	}
	if !strings.Contains(auth, "AKID") {
		t.Error("Authorization should contain access key")
	}
	if !strings.Contains(auth, "us-east-1/bedrock/aws4_request") {
		t.Error("Authorization should contain credential scope")
	}

	amzDate := req.Header.Get("x-amz-date")
	if amzDate == "" {
		t.Error("x-amz-date header should be set")
	}
}

func TestParseEventStream(t *testing.T) {
	// Build a minimal event stream with two events
	var buf bytes.Buffer

	writeEvent := func(eventType string, payload []byte) {
		headers := buildTestHeaders(eventType)
		headersLen := uint32(len(headers))
		payloadLen := uint32(len(payload))
		totalLen := 12 + headersLen + payloadLen + 4

		prelude := make([]byte, 8)
		binary.BigEndian.PutUint32(prelude[0:4], totalLen)
		binary.BigEndian.PutUint32(prelude[4:8], headersLen)
		preludeCRC := crc32.ChecksumIEEE(prelude)

		buf.Write(prelude)
		crcBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(crcBytes, preludeCRC)
		buf.Write(crcBytes)
		buf.Write(headers)
		buf.Write(payload)

		msgCRC := crc32.ChecksumIEEE(buf.Bytes()[buf.Len()-int(totalLen)+12:])
		binary.BigEndian.PutUint32(crcBytes, msgCRC)
		buf.Write(crcBytes)
	}

	writeEvent("contentBlockDelta", []byte(`{"delta":{"text":"Hi"}}`))
	writeEvent("metadata", []byte(`{"usage":{"outputTokens":5}}`))

	var events []struct {
		eventType string
		payload   string
	}

	err := parseEventStream(&buf, func(eventType string, payload []byte) error {
		events = append(events, struct {
			eventType string
			payload   string
		}{eventType, string(payload)})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].eventType != "contentBlockDelta" {
		t.Errorf("event[0].type = %q, want contentBlockDelta", events[0].eventType)
	}
	if events[1].eventType != "metadata" {
		t.Errorf("event[1].type = %q, want metadata", events[1].eventType)
	}
}

func buildTestHeaders(eventType string) []byte {
	// Header: [1B name_len][name][1B type=7][2B value_len][value]
	name := ":event-type"
	var buf bytes.Buffer
	buf.WriteByte(byte(len(name)))
	buf.WriteString(name)
	buf.WriteByte(7) // string type
	valBytes := []byte(eventType)
	binary.Write(&buf, binary.BigEndian, uint16(len(valBytes)))
	buf.Write(valBytes)
	return buf.Bytes()
}
