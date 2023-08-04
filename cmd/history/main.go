package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/buglloc/sowettybot/internal/history"
	"github.com/buglloc/sowettybot/internal/models"
	"github.com/buglloc/sowettybot/internal/renderer"
)

func fatalf(msg string, a ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, fmt.Sprintf("history: %s\n", msg), a...)
	os.Exit(1)
}

func handleLog(entries []models.History) error {
	log, err := renderer.NewHistoryRenderer().Log(entries)
	if err != nil {
		return err
	}

	fmt.Println(log)
	return nil
}

func handleGraph(entries []models.History) error {
	f, err := os.CreateTemp("", "sowetty-history-*.png")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	defer func() { _ = f.Close() }()

	hr := renderer.NewHistoryRenderer()
	cfg := renderer.NewGraphConfig().
		Width(512).
		Height(512)

	startDate, endDate, err := hr.Graph(entries, f, cfg)
	if err != nil {
		return fmt.Errorf("render failed: %w", err)
	}

	fmt.Printf(
		"history %s -> %s rendered into file: %s\n",
		startDate.Format("02 Jan 15:04"), endDate.Format("02 Jan 15:04"), f.Name(),
	)
	return nil
}

func main() {
	var histFile, kind string
	limit := 192
	flag.StringVar(&kind, "kind", "graph", "kind (graph or log)")
	flag.StringVar(&histFile, "file", "", "history file")
	flag.IntVar(&limit, "limit", limit, "history limit")
	flag.Parse()

	if histFile == "" {
		fatalf("--file is required")
	}

	entries, err := history.NewHistory(histFile, limit).Entries(0)
	if err != nil {
		fatalf("read history: %v", err)
	}

	switch kind {
	case "log":
		err = handleLog(entries)
	case "graph":
		err = handleGraph(entries)
	default:
		err = fmt.Errorf("unsupported kind: %s", kind)
	}

	if err != nil {
		fatalf("%v", err)
	}
}
