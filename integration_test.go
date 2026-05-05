package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Jwrede/llmprobe/internal/config"
	"github.com/Jwrede/llmprobe/internal/output"
	"github.com/Jwrede/llmprobe/internal/probe"
)

func TestFullPipelineAllProviders(t *testing.T) {
	openaiServer := mockOpenAI(t, 50*time.Millisecond)
	defer openaiServer.Close()

	anthropicServer := mockAnthropic(t, 80*time.Millisecond)
	defer anthropicServer.Close()

	googleServer := mockGoogle(t, 30*time.Millisecond)
	defer googleServer.Close()

	azureServer := mockAzure(t, 60*time.Millisecond)
	defer azureServer.Close()

	bedrockServer := mockBedrock(t, 70*time.Millisecond)
	defer bedrockServer.Close()

	cfg := &config.Config{
		Defaults: config.Defaults{
			Prompt:      "Hello",
			MaxTokens:   10,
			Timeout:     config.Duration{Duration: 10 * time.Second},
			Concurrency: 5,
		},
		Providers: []config.Provider{
			{
				Name:    "openai",
				APIKey:  "test-key",
				BaseURL: openaiServer.URL,
				Models: []config.Model{
					{
						Name:      "gpt-4o",
						Prompt:    "Hello",
						MaxTokens: 10,
						Thresholds: config.Thresholds{
							MaxTTFT: config.Duration{Duration: 5 * time.Second},
						},
					},
				},
			},
			{
				Name:    "anthropic",
				APIKey:  "test-key",
				BaseURL: anthropicServer.URL,
				Models: []config.Model{
					{
						Name:      "claude-sonnet-4-20250514",
						Prompt:    "Hello",
						MaxTokens: 10,
						Thresholds: config.Thresholds{
							MaxTTFT: config.Duration{Duration: 5 * time.Second},
						},
					},
				},
			},
			{
				Name:    "google",
				APIKey:  "test-key",
				BaseURL: googleServer.URL,
				Models: []config.Model{
					{
						Name:      "gemini-2.0-flash",
						Prompt:    "Hello",
						MaxTokens: 10,
					},
				},
			},
			{
				Name:       "azure",
				APIKey:     "test-key",
				BaseURL:    azureServer.URL,
				APIVersion: "2024-10-21",
				Models: []config.Model{
					{
						Name:      "gpt-4o",
						Prompt:    "Hello",
						MaxTokens: 10,
					},
				},
			},
			{
				Name:      "bedrock",
				AccessKey: "AKID",
				SecretKey: "SECRET",
				Region:    "us-east-1",
				BaseURL:   bedrockServer.URL,
				Models: []config.Model{
					{
						Name:      "anthropic.claude-3-5-sonnet-20241022-v2:0",
						Prompt:    "Hello",
						MaxTokens: 10,
					},
				},
			},
		},
	}

	engine := probe.NewEngine(cfg)
	results, err := engine.RunAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 5 {
		t.Fatalf("got %d results, want 5", len(results))
	}

	for _, r := range results {
		if r.Status == probe.StatusError {
			t.Errorf("%s/%s: unexpected error: %s", r.Provider, r.Model, r.Error)
			continue
		}
		if r.TTFT == 0 {
			t.Errorf("%s/%s: TTFT should be > 0", r.Provider, r.Model)
		}
		if r.TotalLatency == 0 {
			t.Errorf("%s/%s: TotalLatency should be > 0", r.Provider, r.Model)
		}
		if r.TokenCount == 0 {
			t.Errorf("%s/%s: TokenCount should be > 0", r.Provider, r.Model)
		}
		if r.TokensPerSec == 0 {
			t.Errorf("%s/%s: TokensPerSec should be > 0", r.Provider, r.Model)
		}
		if r.Status != probe.StatusHealthy {
			t.Errorf("%s/%s: status = %s, want healthy", r.Provider, r.Model, r.Status)
		}
	}

	var buf bytes.Buffer
	output.WriteTable(&buf, results)
	t.Logf("Table output:\n%s", buf.String())

	buf.Reset()
	if err := output.WriteJSON(&buf, results); err != nil {
		t.Errorf("JSON output error: %v", err)
	}
	t.Logf("JSON output:\n%s", buf.String())
}

func TestThresholdDegraded(t *testing.T) {
	server := mockOpenAI(t, 200*time.Millisecond)
	defer server.Close()

	cfg := &config.Config{
		Defaults: config.Defaults{
			Prompt:      "Hello",
			MaxTokens:   10,
			Timeout:     config.Duration{Duration: 10 * time.Second},
			Concurrency: 1,
		},
		Providers: []config.Provider{
			{
				Name:    "openai",
				APIKey:  "test-key",
				BaseURL: server.URL,
				Models: []config.Model{
					{
						Name:      "gpt-4o",
						Prompt:    "Hello",
						MaxTokens: 10,
						Thresholds: config.Thresholds{
							MaxTTFT: config.Duration{Duration: 10 * time.Millisecond},
						},
					},
				},
			},
		},
	}

	engine := probe.NewEngine(cfg)
	results, err := engine.RunAll()
	if err != nil {
		t.Fatal(err)
	}

	if results[0].Status != probe.StatusDegraded {
		t.Errorf("status = %s, want degraded (TTFT %v should exceed 10ms threshold)",
			results[0].Status, results[0].TTFT)
	}
}

func TestProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.Config{
		Defaults: config.Defaults{
			Prompt:      "Hello",
			MaxTokens:   10,
			Timeout:     config.Duration{Duration: 5 * time.Second},
			Concurrency: 1,
		},
		Providers: []config.Provider{
			{
				Name:    "openai",
				APIKey:  "test-key",
				BaseURL: server.URL,
				Models: []config.Model{
					{Name: "gpt-4o", Prompt: "Hello", MaxTokens: 10},
				},
			},
		},
	}

	engine := probe.NewEngine(cfg)
	results, err := engine.RunAll()
	if err != nil {
		t.Fatal(err)
	}

	if results[0].Status != probe.StatusError {
		t.Errorf("status = %s, want error", results[0].Status)
	}
}

// Mock servers

func mockOpenAI(t *testing.T, ttftDelay time.Duration) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		f := w.(http.Flusher)
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\n")
		f.Flush()
		time.Sleep(ttftDelay)
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n")
		f.Flush()
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\" there\"}}]}\n\n")
		f.Flush()
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\" friend\"}}]}\n\n")
		f.Flush()
		fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"completion_tokens\":3}}\n\n")
		f.Flush()
		fmt.Fprint(w, "data: [DONE]\n\n")
		f.Flush()
	}))
}

func mockAnthropic(t *testing.T, ttftDelay time.Duration) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		f := w.(http.Flusher)
		fmt.Fprint(w, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_test\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-sonnet-4-20250514\",\"usage\":{\"input_tokens\":10,\"output_tokens\":0}}}\n\n")
		f.Flush()
		fmt.Fprint(w, "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
		f.Flush()
		fmt.Fprint(w, "event: ping\ndata: {\"type\":\"ping\"}\n\n")
		f.Flush()
		time.Sleep(ttftDelay)
		fmt.Fprint(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n")
		f.Flush()
		fmt.Fprint(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\" there\"}}\n\n")
		f.Flush()
		fmt.Fprint(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\" friend\"}}\n\n")
		f.Flush()
		fmt.Fprint(w, "event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n")
		f.Flush()
		fmt.Fprint(w, "event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":3}}\n\n")
		f.Flush()
		fmt.Fprint(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
		f.Flush()
	}))
}

func mockGoogle(t *testing.T, ttftDelay time.Duration) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		f := w.(http.Flusher)
		time.Sleep(ttftDelay)
		fmt.Fprint(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Hello\"}]}}]}\n\n")
		f.Flush()
		fmt.Fprint(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\" there\"}]}}]}\n\n")
		f.Flush()
		fmt.Fprint(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\" friend\"}]}}],\"usageMetadata\":{\"candidatesTokenCount\":3}}\n\n")
		f.Flush()
	}))
}

func mockAzure(t *testing.T, ttftDelay time.Duration) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		f := w.(http.Flusher)
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\n")
		f.Flush()
		time.Sleep(ttftDelay)
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n")
		f.Flush()
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\" there\"}}]}\n\n")
		f.Flush()
		fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"completion_tokens\":2}}\n\n")
		f.Flush()
		fmt.Fprint(w, "data: [DONE]\n\n")
		f.Flush()
	}))
}

func mockBedrock(t *testing.T, ttftDelay time.Duration) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.amazon.eventstream")
		writeAWSEvent(w, "messageStart", []byte(`{"role":"assistant"}`))
		writeAWSEvent(w, "contentBlockStart", []byte(`{"contentBlockIndex":0,"start":{"text":""}}`))
		time.Sleep(ttftDelay)
		writeAWSEvent(w, "contentBlockDelta", []byte(`{"contentBlockIndex":0,"delta":{"text":"Hello"}}`))
		writeAWSEvent(w, "contentBlockDelta", []byte(`{"contentBlockIndex":0,"delta":{"text":" there"}}`))
		writeAWSEvent(w, "contentBlockDelta", []byte(`{"contentBlockIndex":0,"delta":{"text":" friend"}}`))
		writeAWSEvent(w, "contentBlockStop", []byte(`{"contentBlockIndex":0}`))
		writeAWSEvent(w, "messageStop", []byte(`{"stopReason":"end_turn"}`))
		writeAWSEvent(w, "metadata", []byte(`{"usage":{"inputTokens":10,"outputTokens":3,"totalTokens":13}}`))
	}))
}

func writeAWSEvent(w http.ResponseWriter, eventType string, payload []byte) {
	headers := buildEventHeaders(eventType)
	headersLen := uint32(len(headers))
	payloadLen := uint32(len(payload))
	totalLen := 12 + headersLen + payloadLen + 4

	prelude := make([]byte, 8)
	binary.BigEndian.PutUint32(prelude[0:4], totalLen)
	binary.BigEndian.PutUint32(prelude[4:8], headersLen)
	preludeCRC := crc32.ChecksumIEEE(prelude)

	var msg bytes.Buffer
	msg.Write(prelude)
	crcBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBuf, preludeCRC)
	msg.Write(crcBuf)
	msg.Write(headers)
	msg.Write(payload)

	msgCRC := crc32.ChecksumIEEE(msg.Bytes()[12:])
	binary.BigEndian.PutUint32(crcBuf, msgCRC)
	msg.Write(crcBuf)

	w.Write(msg.Bytes())
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func buildEventHeaders(eventType string) []byte {
	name := ":event-type"
	var buf bytes.Buffer
	buf.WriteByte(byte(len(name)))
	buf.WriteString(name)
	buf.WriteByte(7)
	binary.Write(&buf, binary.BigEndian, uint16(len(eventType)))
	buf.WriteString(eventType)
	return buf.Bytes()
}
