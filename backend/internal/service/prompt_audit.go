package service

import (
	"bufio"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/new-api-tools/backend/internal/config"
)

var promptAuditAppendMu sync.Mutex

type PromptAuditEvent struct {
	Source          string   `json:"source"`
	RequestID       string   `json:"request_id"`
	UserID          int      `json:"user_id"`
	TokenID         int      `json:"token_id"`
	Model           string   `json:"model"`
	Group           string   `json:"group"`
	ChannelID       int      `json:"channel_id"`
	Prompt          string   `json:"prompt"`
	PromptSHA256    string   `json:"prompt_sha256"`
	PromptTruncated bool     `json:"prompt_truncated"`
	MatchedKeywords []string `json:"matched_keywords"`
	CreatedAt       int64    `json:"created_at"`
	ReceivedAt      int64    `json:"received_at"`
}

type PromptAuditEventList struct {
	Items  []PromptAuditEvent `json:"items"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
	Path   string             `json:"path"`
}

func PromptAuditSecretConfigured() bool {
	return strings.TrimSpace(os.Getenv("PROMPT_AUDIT_SECRET")) != ""
}

func VerifyPromptAuditSecret(got string) bool {
	expected := strings.TrimSpace(os.Getenv("PROMPT_AUDIT_SECRET"))
	got = strings.TrimSpace(got)
	if expected == "" || got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

func SavePromptAuditEvent(event PromptAuditEvent) error {
	if strings.TrimSpace(event.Prompt) == "" {
		return errors.New("prompt is required")
	}
	if event.CreatedAt == 0 {
		event.CreatedAt = time.Now().Unix()
	}
	event.ReceivedAt = time.Now().Unix()
	if strings.TrimSpace(event.Source) == "" {
		event.Source = "new-api"
	}

	path, err := promptAuditEventPath()
	if err != nil {
		return err
	}

	promptAuditAppendMu.Lock()
	defer promptAuditAppendMu.Unlock()

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(event)
}

func ListPromptAuditEvents(limit, offset int) (PromptAuditEventList, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	path, err := promptAuditEventPath()
	if err != nil {
		return PromptAuditEventList{}, err
	}

	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return PromptAuditEventList{
			Items:  []PromptAuditEvent{},
			Total:  0,
			Limit:  limit,
			Offset: offset,
			Path:   path,
		}, nil
	}
	if err != nil {
		return PromptAuditEventList{}, err
	}
	defer f.Close()

	events, err := readPromptAuditJSONL(f)
	if err != nil {
		return PromptAuditEventList{}, err
	}

	reversePromptAuditEvents(events)
	total := len(events)
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	return PromptAuditEventList{
		Items:  events[start:end],
		Total:  total,
		Limit:  limit,
		Offset: offset,
		Path:   path,
	}, nil
}

func readPromptAuditJSONL(r io.Reader) ([]PromptAuditEvent, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	events := make([]PromptAuditEvent, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event PromptAuditEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func reversePromptAuditEvents(events []PromptAuditEvent) {
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}
}

func promptAuditEventPath() (string, error) {
	cfg := config.Get()
	dir := filepath.Join(cfg.DataDir, "prompt_audit")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "events.jsonl"), nil
}
