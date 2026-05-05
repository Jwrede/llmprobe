package provider

import (
	"bufio"
	"io"
	"strings"
	"time"
)

type Provider interface {
	Name() string
	Probe(ctx ProviderContext) (Result, error)
}

type ProviderContext struct {
	Model     string
	Prompt    string
	MaxTokens int
	Timeout   time.Duration
}

type Result struct {
	TTFT         time.Duration
	TotalLatency time.Duration
	TokenCount   int
}

func ParseSSEStream(r io.Reader, onEvent func(eventType string, data []byte) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var currentEvent string
	var currentData []byte

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		switch {
		case strings.HasPrefix(line, "event: "):
			currentEvent = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "event:"):
			currentEvent = strings.TrimPrefix(line, "event:")
		case strings.HasPrefix(line, "data: "):
			currentData = []byte(strings.TrimPrefix(line, "data: "))
		case strings.HasPrefix(line, "data:"):
			currentData = []byte(strings.TrimPrefix(line, "data:"))
		case line == "":
			if currentData != nil {
				if err := onEvent(currentEvent, currentData); err != nil {
					return err
				}
			}
			currentEvent = ""
			currentData = nil
		}
	}

	if currentData != nil {
		if err := onEvent(currentEvent, currentData); err != nil {
			return err
		}
	}

	return scanner.Err()
}
