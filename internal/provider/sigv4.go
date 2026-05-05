package provider

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

func signV4(req *http.Request, body []byte, region, service, accessKey, secretKey string) {
	now := time.Now().UTC()
	datestamp := now.Format("20060102")
	amzdate := now.Format("20060102T150405Z")

	req.Header.Set("x-amz-date", amzdate)

	payloadHash := sha256Hex(body)
	req.Header.Set("x-amz-content-sha256", payloadHash)

	signedHeaders, canonicalHeaders := buildCanonicalHeaders(req)

	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.Path,
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", datestamp, region, service)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzdate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := deriveSigningKey(secretKey, datestamp, region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey, credentialScope, signedHeaders, signature,
	))
}

func buildCanonicalHeaders(req *http.Request) (signedHeaders, canonicalHeaders string) {
	headers := make(map[string]string)
	var keys []string
	for k, v := range req.Header {
		lower := strings.ToLower(k)
		if lower == "host" || lower == "content-type" || strings.HasPrefix(lower, "x-amz-") {
			headers[lower] = strings.TrimSpace(v[0])
			keys = append(keys, lower)
		}
	}
	headers["host"] = req.URL.Host
	keys = append(keys, "host")

	sort.Strings(keys)
	seen := make(map[string]bool)
	var uniqueKeys []string
	for _, k := range keys {
		if !seen[k] {
			seen[k] = true
			uniqueKeys = append(uniqueKeys, k)
		}
	}

	var canonical strings.Builder
	for _, k := range uniqueKeys {
		canonical.WriteString(k)
		canonical.WriteString(":")
		canonical.WriteString(headers[k])
		canonical.WriteString("\n")
	}

	return strings.Join(uniqueKeys, ";"), canonical.String()
}

func deriveSigningKey(secretKey, datestamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(datestamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// parseEventStream reads AWS binary event stream frames from r.
// Each frame: [4B total_len][4B headers_len][4B prelude_crc][headers][payload][4B msg_crc]
func parseEventStream(r io.Reader, onEvent func(eventType string, payload []byte) error) error {
	for {
		prelude := make([]byte, 12)
		if _, err := io.ReadFull(r, prelude); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			return err
		}

		totalLen := beUint32(prelude[0:4])
		headersLen := beUint32(prelude[4:8])

		remaining := make([]byte, totalLen-12)
		if _, err := io.ReadFull(r, remaining); err != nil {
			return err
		}

		headerBytes := remaining[:headersLen]
		payloadLen := totalLen - 12 - headersLen - 4
		payload := remaining[headersLen : headersLen+payloadLen]

		eventType := extractEventType(headerBytes)

		if err := onEvent(eventType, payload); err != nil {
			return err
		}
	}
}

func extractEventType(headers []byte) string {
	pos := 0
	for pos < len(headers) {
		if pos >= len(headers) {
			break
		}
		nameLen := int(headers[pos])
		pos++
		if pos+nameLen > len(headers) {
			break
		}
		name := string(headers[pos : pos+nameLen])
		pos += nameLen

		if pos >= len(headers) {
			break
		}
		valueType := headers[pos]
		pos++

		switch valueType {
		case 7: // string
			if pos+2 > len(headers) {
				return ""
			}
			valLen := int(beUint16(headers[pos : pos+2]))
			pos += 2
			if pos+valLen > len(headers) {
				return ""
			}
			val := string(headers[pos : pos+valLen])
			pos += valLen
			if name == ":event-type" {
				return val
			}
		default:
			return ""
		}
	}
	return ""
}

func beUint32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

func beUint16(b []byte) uint16 {
	return uint16(b[0])<<8 | uint16(b[1])
}
