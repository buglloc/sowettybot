package history

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/icza/backscanner"
	"github.com/rs/zerolog/log"

	"github.com/buglloc/sowettybot/internal/models"
)

type History struct {
	storeFile   string
	limit       int
	mu          sync.Mutex
	lastModTime time.Time
	lastEntries []models.History
}

func NewHistory(storeFile string, limit int) *History {
	return &History{
		storeFile: storeFile,
		limit:     limit,
	}
}

func (h *History) Entries(limit int) ([]models.History, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.lockedEntries(limit)
}

func (h *History) lockedEntries(limit int) ([]models.History, error) {
	if h.storeFile == "" {
		return nil, nil
	}

	f, err := os.Open(h.storeFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open history file: %w", err)
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("unable to get history stat: %w", err)
	}
	defer func() { _ = f.Close() }()

	if h.lastModTime == fi.ModTime() {
		return limitedEntries(h.lastEntries, limit), nil
	}

	scanner := backscanner.New(f, int(fi.Size()))
	var lines []string
	for {
		line, _, err := scanner.Line()
		if err != nil {
			break
		}

		if line == "" {
			continue
		}

		lines = append(lines, line)
		if len(lines) > h.limit {
			break
		}
	}

	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}

	h.lastModTime = fi.ModTime()
	h.lastEntries = h.lastEntries[:0]
	expectedValues := 0
	for _, line := range lines {
		entry, err := parseEntry(line)
		if err != nil {
			log.Error().Err(err).Str("line", line).Msg("invalid history line")
			continue
		}

		if expectedValues == 0 {
			if len(entry.Names) != len(entry.Values) {
				err := fmt.Errorf("invalid data: %d (names) != %d (values)", len(entry.Names), len(entry.Values))
				log.Error().Err(err).Str("line", line).Msg("invalid history line")
				continue
			}

			expectedValues = len(entry.Names)
		} else {
			if len(entry.Names) != expectedValues {
				err := fmt.Errorf("invalid names: %d (actual) != %d (expected)", len(entry.Names), expectedValues)
				log.Error().Err(err).Str("line", line).Msg("invalid history line")
				continue
			}

			if len(entry.Values) != expectedValues {
				err := fmt.Errorf("invalid values: %d (actual) != %d (expected)", len(entry.Values), expectedValues)
				log.Error().Err(err).Str("line", line).Msg("invalid history line")
				continue
			}
		}

		h.lastEntries = append(h.lastEntries, entry)
	}

	return limitedEntries(h.lastEntries, limit), nil
}

func parseEntry(in string) (models.History, error) {
	whenStr, rest := in[:len(time.RFC822)], in[len(time.RFC822)+1:]
	when, err := time.Parse(time.RFC822, whenStr)
	if err != nil {
		return models.History{}, fmt.Errorf("invalid date %q: %w", whenStr, err)
	}

	fields := strings.Fields(rest)
	var names []string
	var values []float64
	for _, kv := range fields {
		data := strings.SplitN(kv, "=", 2)
		name := strings.TrimSpace(data[0])
		value, err := strconv.ParseFloat(strings.TrimSpace(data[1]), 64)
		if err != nil {
			return models.History{}, fmt.Errorf("invalid field %q: %w", whenStr, err)
		}

		names = append(names, name)
		values = append(values, value)
	}

	return models.History{
		When:   when.Local(),
		Names:  names,
		Values: values,
	}, nil
}

func limitedEntries(entries []models.History, limit int) []models.History {
	if limit == 0 || len(entries) < limit {
		limit = len(entries)
	}

	return entries[len(entries)-limit:]
}
