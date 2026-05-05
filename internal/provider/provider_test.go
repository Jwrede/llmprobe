package provider

import (
	"strings"
	"testing"
)

func TestParseSSEStreamDataOnly(t *testing.T) {
	input := "data: {\"msg\":\"hello\"}\n\ndata: {\"msg\":\"world\"}\n\ndata: [DONE]\n\n"
	var events []string

	err := ParseSSEStream(strings.NewReader(input), func(eventType string, data []byte) error {
		events = append(events, string(data))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	if events[0] != `{"msg":"hello"}` {
		t.Errorf("event[0] = %q", events[0])
	}
	if events[2] != "[DONE]" {
		t.Errorf("event[2] = %q", events[2])
	}
}

func TestParseSSEStreamNamedEvents(t *testing.T) {
	input := "event: message_start\ndata: {\"type\":\"message_start\"}\n\nevent: content_block_delta\ndata: {\"delta\":{\"text\":\"Hi\"}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	type ev struct {
		eventType string
		data      string
	}
	var events []ev

	err := ParseSSEStream(strings.NewReader(input), func(eventType string, data []byte) error {
		events = append(events, ev{eventType, string(data)})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	if events[0].eventType != "message_start" {
		t.Errorf("event[0].type = %q, want message_start", events[0].eventType)
	}
	if events[1].eventType != "content_block_delta" {
		t.Errorf("event[1].type = %q, want content_block_delta", events[1].eventType)
	}
}

func TestParseSSEStreamNoTrailingNewline(t *testing.T) {
	input := "data: final"
	var events []string

	err := ParseSSEStream(strings.NewReader(input), func(eventType string, data []byte) error {
		events = append(events, string(data))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
}

func TestParseSSEStreamEmpty(t *testing.T) {
	var events []string
	err := ParseSSEStream(strings.NewReader(""), func(eventType string, data []byte) error {
		events = append(events, string(data))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("got %d events, want 0", len(events))
	}
}
