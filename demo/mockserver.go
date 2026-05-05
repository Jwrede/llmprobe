package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"math/rand"
	"net/http"
	"os"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", handleOpenAI)
	mux.HandleFunc("/v1/messages", handleAnthropic)
	mux.HandleFunc("/v1beta/models/", handleGoogle)
	mux.HandleFunc("/openai/deployments/", handleAzure)
	mux.HandleFunc("/model/", handleBedrock)

	port := "9119"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	fmt.Fprintf(os.Stderr, "Mock LLM server on :%s\n", port)
	http.ListenAndServe(":"+port, mux)
}

func jitter(base time.Duration) time.Duration {
	noise := time.Duration(rand.Int63n(int64(base / 5)))
	return base + noise - base/10
}

func handleOpenAI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	f := w.(http.Flusher)

	fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\n")
	f.Flush()

	time.Sleep(jitter(280 * time.Millisecond))
	tokens := []string{"The", " weather", " is", " sunny", " with", " clear", " skies", " today", "."}
	for _, tok := range tokens {
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", tok)
		f.Flush()
		time.Sleep(jitter(35 * time.Millisecond))
	}

	fmt.Fprintf(w, "data: {\"choices\":[],\"usage\":{\"completion_tokens\":%d}}\n\n", len(tokens))
	f.Flush()
	fmt.Fprint(w, "data: [DONE]\n\n")
	f.Flush()
}

func handleAnthropic(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	f := w.(http.Flusher)

	fmt.Fprint(w, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_demo\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-sonnet-4-20250514\",\"usage\":{\"input_tokens\":12,\"output_tokens\":0}}}\n\n")
	f.Flush()
	fmt.Fprint(w, "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
	f.Flush()

	time.Sleep(jitter(410 * time.Millisecond))
	tokens := []string{"Today", "'s", " forecast", " calls", " for", " partly", " cloudy", " skies", " with", " mild", " temperatures", "."}
	for i, tok := range tokens {
		fmt.Fprintf(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":%q}}\n\n", tok)
		f.Flush()
		_ = i
		time.Sleep(jitter(28 * time.Millisecond))
	}

	fmt.Fprint(w, "event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n")
	f.Flush()
	fmt.Fprintf(w, "event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":%d}}\n\n", len(tokens))
	f.Flush()
	fmt.Fprint(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	f.Flush()
}

func handleGoogle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	f := w.(http.Flusher)

	time.Sleep(jitter(180 * time.Millisecond))
	tokens := []string{"Expect", " warm", " and", " humid", " conditions", " throughout", " the", " afternoon", "."}
	for i, tok := range tokens {
		if i < len(tokens)-1 {
			fmt.Fprintf(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":%q}]}}]}\n\n", tok)
		} else {
			fmt.Fprintf(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":%q}]}}],\"usageMetadata\":{\"candidatesTokenCount\":%d}}\n\n", tok, len(tokens))
		}
		f.Flush()
		time.Sleep(jitter(22 * time.Millisecond))
	}
}

func handleAzure(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	f := w.(http.Flusher)

	fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\n")
	f.Flush()

	time.Sleep(jitter(320 * time.Millisecond))
	tokens := []string{"Light", " rain", " expected", " this", " evening", " with", " temperatures", " dropping", "."}
	for _, tok := range tokens {
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", tok)
		f.Flush()
		time.Sleep(jitter(30 * time.Millisecond))
	}

	fmt.Fprintf(w, "data: {\"choices\":[],\"usage\":{\"completion_tokens\":%d}}\n\n", len(tokens))
	f.Flush()
	fmt.Fprint(w, "data: [DONE]\n\n")
	f.Flush()
}

func handleBedrock(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/vnd.amazon.eventstream")

	writeAWSEvent(w, "messageStart", []byte(`{"role":"assistant"}`))
	writeAWSEvent(w, "contentBlockStart", []byte(`{"contentBlockIndex":0,"start":{"text":""}}`))

	time.Sleep(jitter(350 * time.Millisecond))
	tokens := []string{"Overcast", " skies", " with", " a", " chance", " of", " afternoon", " showers", "."}
	for _, tok := range tokens {
		writeAWSEvent(w, "contentBlockDelta", []byte(fmt.Sprintf(`{"contentBlockIndex":0,"delta":{"text":%q}}`, tok)))
		time.Sleep(jitter(32 * time.Millisecond))
	}

	writeAWSEvent(w, "contentBlockStop", []byte(`{"contentBlockIndex":0}`))
	writeAWSEvent(w, "messageStop", []byte(`{"stopReason":"end_turn"}`))
	writeAWSEvent(w, "metadata", []byte(fmt.Sprintf(`{"usage":{"inputTokens":12,"outputTokens":%d,"totalTokens":%d}}`, len(tokens), 12+len(tokens))))
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
