package store

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/balazscsaba2006/specflow/internal/models"
)

// AppendLog sets Timestamp if zero, marshals the entry to JSON, and appends
// one line to log.jsonl.
func (s *Store) AppendLog(entry models.LogEntry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC().Truncate(time.Second)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling log entry: %w", err)
	}

	f, err := os.OpenFile(s.LogFile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing log entry: %w", err)
	}

	return nil
}

// ReadLog reads all lines from log.jsonl and returns the last N entries.
// If last <= 0, returns all entries.
func (s *Store) ReadLog(last int) ([]models.LogEntry, error) {
	data, err := os.ReadFile(s.LogFile())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading log file: %w", err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return nil, nil
	}

	lines := strings.Split(content, "\n")
	entries := make([]models.LogEntry, 0, len(lines))
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry models.LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("parsing log line %d: %w", i+1, err)
		}
		entries = append(entries, entry)
	}

	if last > 0 && last < len(entries) {
		entries = entries[len(entries)-last:]
	}

	return entries, nil
}
