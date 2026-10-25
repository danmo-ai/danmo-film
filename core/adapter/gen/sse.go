package gen

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/danmo-ai/danmo-film/core/port"
)

func readSSE(r io.Reader, ch chan<- port.SSEEvent) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	event := "message"
	var data []string
	flush := func() {
		if len(data) == 0 {
			return
		}
		ch <- port.SSEEvent{Event: event, Data: strings.Join(data, "\n")}
		event = "message"
		data = nil
	}
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			event = strings.TrimSpace(line[6:])
			if event == "" {
				event = "message"
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			data = append(data, strings.TrimLeft(line[5:], " "))
			continue
		}
		if line == "" {
			flush()
		}
	}
	flush()
}

func pollToSSE(ctx context.Context, poll func() (map[string]any, error)) (<-chan port.SSEEvent, error) {
	ch := make(chan port.SSEEvent)
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			task, err := poll()
			if err != nil {
				ch <- port.SSEEvent{Event: "error", Data: `{"error":` + jsonQuote(err.Error()) + `}`}
				return
			}
			status := strings.ToLower(strVal(task["status"]))
			raw, _ := json.Marshal(task)
			ch <- port.SSEEvent{Event: "progress", Data: string(raw)}
			if status == "completed" || status == "succeeded" || status == "success" {
				ch <- port.SSEEvent{Event: "result", Data: string(raw)}
				ch <- port.SSEEvent{Event: "done", Data: string(raw)}
				return
			}
			if status == "failed" || status == "error" || status == "cancelled" || status == "canceled" {
				ch <- port.SSEEvent{Event: "error", Data: string(raw)}
				return
			}
			sleepPoll()
		}
	}()
	return ch, nil
}
