package client

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/crypto"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/models"
)

// multipartBuilder constructs multipart/mixed request bodies.
type multipartBuilder struct {
	encryptor         *crypto.Encryptor
	keyUUID           string
	enableCompression bool
}

func (b *multipartBuilder) build(entries []models.LogEntry) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, entry := range entries {
		payloadType := entry.PayloadType
		if payloadType == 0 {
			payloadType = models.DefaultPayloadType(entry.EntryType)
		}

		headers := make(textproto.MIMEHeader)
		headers.Set("Content-Type", "application/octet-stream")
		headers.Set("X-LF-Entry-Type", strconv.Itoa(entry.EntryType))
		headers.Set("X-LF-Payload-Type", strconv.Itoa(payloadType))

		if !entry.Timestamp.IsZero() {
			headers.Set("X-LF-Timestamp", entry.Timestamp.UTC().Format(time.RFC3339Nano))
		}
		if len(entry.SearchTokens) > 0 {
			headers.Set("X-LF-Search-Tokens", strings.Join(entry.SearchTokens, ","))
		}

		var partBody []byte

		if models.EntryTypeRequiresEncryption(entry.EntryType) {
			raw, err := b.encryptor.EncryptRaw([]byte(entry.Message), b.enableCompression)
			if err != nil {
				return nil, "", fmt.Errorf("encrypt failed: %w", err)
			}
			headers.Set("X-LF-Key-ID", b.keyUUID)
			headers.Set("X-LF-Nonce", base64.StdEncoding.EncodeToString(raw.Nonce))
			partBody = raw.Ciphertext
		} else {
			// Type 7: compress only
			if b.enableCompression {
				compressed, err := crypto.GzipCompress([]byte(entry.Message))
				if err != nil {
					return nil, "", fmt.Errorf("compress failed: %w", err)
				}
				partBody = compressed
			} else {
				partBody = []byte(entry.Message)
			}
		}

		part, err := writer.CreatePart(headers)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create MIME part: %w", err)
		}
		if _, err := part.Write(partBody); err != nil {
			return nil, "", fmt.Errorf("failed to write MIME part: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return body, "multipart/mixed; boundary=" + writer.Boundary(), nil
}
