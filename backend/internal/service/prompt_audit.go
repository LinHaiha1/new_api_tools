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
var promptAuditConfigMu sync.Mutex

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

type PromptAuditConfig struct {
	Enabled   bool     `json:"enabled"`
	Version   int64    `json:"version"`
	Keywords  []string `json:"keywords"`
	UpdatedAt int64    `json:"updated_at"`
}

type PromptAuditConfigInput struct {
	Enabled  bool     `json:"enabled"`
	Keywords []string `json:"keywords"`
	Raw      string   `json:"raw"`
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

func GetPromptAuditConfig() (PromptAuditConfig, error) {
	path, err := promptAuditConfigPath()
	if err != nil {
		return PromptAuditConfig{}, err
	}

	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return PromptAuditConfig{
			Enabled:   false,
			Version:   0,
			Keywords:  []string{},
			UpdatedAt: 0,
		}, nil
	}
	if err != nil {
		return PromptAuditConfig{}, err
	}
	defer f.Close()

	var cfg PromptAuditConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return PromptAuditConfig{}, err
	}
	cfg.Keywords = normalizePromptAuditKeywords(cfg.Keywords)
	return cfg, nil
}

func SavePromptAuditConfig(input PromptAuditConfigInput) (PromptAuditConfig, error) {
	keywords := normalizePromptAuditKeywords(input.Keywords)
	if strings.TrimSpace(input.Raw) != "" {
		keywords = normalizePromptAuditKeywords(append(keywords, splitPromptAuditKeywords(input.Raw)...))
	}

	promptAuditConfigMu.Lock()
	defer promptAuditConfigMu.Unlock()

	current, _ := GetPromptAuditConfig()
	now := time.Now().Unix()
	cfg := PromptAuditConfig{
		Enabled:   input.Enabled,
		Version:   current.Version + 1,
		Keywords:  keywords,
		UpdatedAt: now,
	}
	if cfg.Version <= 0 {
		cfg.Version = now
	}

	path, err := promptAuditConfigPath()
	if err != nil {
		return PromptAuditConfig{}, err
	}

	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return PromptAuditConfig{}, err
	}
	encErr := json.NewEncoder(f).Encode(cfg)
	closeErr := f.Close()
	if encErr != nil {
		_ = os.Remove(tmp)
		return PromptAuditConfig{}, encErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return PromptAuditConfig{}, closeErr
	}
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return PromptAuditConfig{}, err
	}

	return cfg, nil
}

func splitPromptAuditKeywords(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.NewReplacer(",", "\n", "，", "\n", ";", "\n", "；", "\n").Replace(raw)
	return strings.Split(raw, "\n")
}

func normalizePromptAuditKeywords(input []string) []string {
	result := make([]string, 0, len(input))
	seen := make(map[string]struct{})
	for _, item := range input {
		keyword := strings.ToLower(strings.TrimSpace(item))
		if keyword == "" {
			continue
		}
		if _, ok := seen[keyword]; ok {
			continue
		}
		seen[keyword] = struct{}{}
		result = append(result, keyword)
	}
	return result
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

func promptAuditConfigPath() (string, error) {
	cfg := config.Get()
	dir := filepath.Join(cfg.DataDir, "prompt_audit")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}
