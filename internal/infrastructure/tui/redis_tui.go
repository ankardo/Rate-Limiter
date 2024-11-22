package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"github.com/rivo/tview"
)

func main() {
	loadEnv()

	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")

	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := tview.NewApplication()
	envTable := tview.NewTable()
	redisTable := tview.NewTable()
	logView := tview.NewTextView()
	logView.
		SetScrollable(true).
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetBorder(true).
		SetTitle("Logs")

	tooltip := tview.NewTextView()
	tooltip.
		SetText("[yellow]Press 1 to CURL IP, 2 to CURL Token, 3 Clear the Log, 4 to Exit. Use Up/Down/PageUp/PageDown to scroll logs.[white]").
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	lineCounter := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight)

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tooltip, 1, 0, false).
		AddItem(
			tview.NewFlex().
				SetDirection(tview.FlexColumn).
				AddItem(envTable, 0, 3, false).
				AddItem(redisTable, 0, 1, false),
			0, 3, true).
		AddItem(
			tview.NewFlex().
				SetDirection(tview.FlexRow).
				AddItem(logView, 0, 4, true).
				AddItem(lineCounter, 1, 0, false),
			0, 5, false)

	scrollOffset := 0

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case '1':
			curlRequest(ctx, logView, "ip")
		case '2':
			curlRequest(ctx, logView, "token")
		case '3':
			clearLogs(logView, lineCounter)
		case '4':
			cancel()
			app.Stop()

		}

		switch event.Key() {
		case tcell.KeyUp:
			if scrollOffset > 0 {
				scrollOffset--
			}
			logView.ScrollTo(scrollOffset, 0)
		case tcell.KeyDown:
			scrollOffset++
			logView.ScrollTo(scrollOffset, 0)
		case tcell.KeyPgUp:
			if scrollOffset > 10 {
				scrollOffset -= 10
			} else {
				scrollOffset = 0
			}
			logView.ScrollTo(scrollOffset, 0)
		case tcell.KeyPgDn:
			scrollOffset += 10
			logView.ScrollTo(scrollOffset, 0)
		}

		lineCounter.SetText("Line " + strconv.Itoa(scrollOffset+1) + "/" + strconv.Itoa(countLines(logView)))
		return event
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				updateEnvTable(envTable)
				updateRedisTable(ctx, client, redisTable)
				app.Draw()
				time.Sleep(1 * time.Second)
			}
		}
	}()

	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
		<-signalChan
		cancel()
		app.Stop()
	}()

	app.SetRoot(layout, true)
	if err := app.Run(); err != nil {
		os.Exit(1)
	}
}

func loadEnv() {
	_ = godotenv.Load()
}

func updateEnvTable(envTable *tview.Table) {
	envVars := []struct {
		Key   string
		Value string
	}{
		{"RATE_LIMITER_ADDR", os.Getenv("RATE_LIMITER_ADDR")},
		{"BLOCK_DURATION_SECONDS", os.Getenv("BLOCK_DURATION_SECONDS")},
		{"TOKEN_MAX_REQUESTS", os.Getenv("TOKEN_MAX_REQUESTS")},
		{"USE_MEMORY_STORE", os.Getenv("USE_MEMORY_STORE")},
		{"MAX_REQUESTS_PER_SECOND", os.Getenv("MAX_REQUESTS_PER_SECOND")},
		{"REDIS_ADDR", os.Getenv("REDIS_ADDR")},
		{"REDIS_PASSWORD", os.Getenv("REDIS_PASSWORD")},
		{"TTL_EXPIRATION_SECONDS", os.Getenv("TTL_EXPIRATION_SECONDS")},
	}

	envTable.Clear()
	envTable.SetTitle("Environment Variables").SetBorder(true)
	for i, env := range envVars {
		envTable.SetCell(i, 0, tview.NewTableCell(env.Key).SetAlign(tview.AlignLeft))
		envTable.SetCell(i, 1, tview.NewTableCell(env.Value).SetAlign(tview.AlignCenter))
	}
}

func updateRedisTable(ctx context.Context, client *redis.Client, redisTable *tview.Table) {
	keys, err := client.Keys(ctx, "*").Result()
	if err != nil {
		return
	}

	redisTable.Clear()
	redisTable.SetTitle("Redis Rate Limiter Data").SetBorder(true)
	redisTable.SetCell(0, 0, tview.NewTableCell("Key").SetAlign(tview.AlignCenter))
	redisTable.SetCell(0, 1, tview.NewTableCell("Count").SetAlign(tview.AlignCenter))
	redisTable.SetCell(0, 2, tview.NewTableCell("TTL (s)").SetAlign(tview.AlignCenter))
	redisTable.SetCell(0, 3, tview.NewTableCell("Type").SetAlign(tview.AlignCenter))

	sort.Strings(keys)
	for i, key := range keys {
		count, err := client.Get(ctx, key).Result()
		if err != nil {
			count = "N/A"
		}

		ttl, err := client.TTL(ctx, key).Result()
		if err != nil {
			ttl = -1
		}

		redisTable.SetCell(i+1, 0, tview.NewTableCell(key).SetAlign(tview.AlignLeft))
		redisTable.SetCell(i+1, 1, tview.NewTableCell(count).SetAlign(tview.AlignCenter))
		redisTable.SetCell(i+1, 2, tview.NewTableCell(formatTTL(ttl)).SetAlign(tview.AlignCenter))
		if strings.HasPrefix(key, "token:") {
			redisTable.SetCell(i+1, 3, tview.NewTableCell("Token").SetAlign(tview.AlignCenter))
		} else {
			redisTable.SetCell(i+1, 3, tview.NewTableCell("IP").SetAlign(tview.AlignCenter))
		}
	}
}

func runE2ETests(ctx context.Context, logView *tview.TextView, lineCounter *tview.TextView) {
	appendLogs(logView, "Starting E2E Tests...\n")
	startCmd := exec.Command("docker", "logs", "rate-limiter")
	startCmd.Stdout = logWriter(logView, "[rate-limiter]")
	startCmd.Stderr = logWriter(logView, "[rate-limiter]")
	_ = startCmd.Run()
	startCmd = exec.Command("docker", "start", "-ai", "rate-limiter-tests")
	startCmd.Stdout = logWriter(logView, "[rate-limiter-tests]")
	startCmd.Stderr = logWriter(logView, "[rate-limiter-tests]")
	if err := startCmd.Run(); err != nil {
		appendLogs(logView, "Error running tests: "+err.Error()+"\n")
	}
	appendLogs(logView, "E2E Tests completed.\n")

	lineCounter.SetText("Line " + strconv.Itoa(countLines(logView)) + "/" + strconv.Itoa(countLines(logView)))
}

func curlRequest(ctx context.Context, logView *tview.TextView, keyType string) {
	var curlCmd *exec.Cmd
	host := os.Getenv("RATE_LIMITER_ADDR")
	if host == "" {
		host = "localhost:8080"
	}

	if keyType == "ip" {
		curlCmd = exec.Command("curl", "-X", "GET", fmt.Sprintf("http://%s/ip?ip=192.168.1.1", host))
	} else {
		curlCmd = exec.Command("curl", "-X", "GET", fmt.Sprintf("http://%s/token?token=valid-token", host))
	}
	appendLogs(logView, fmt.Sprintf("Running CURL command: %s\n", strings.Join(curlCmd.Args, " ")))
	curlCmd.Stdout = logWriter(logView, fmt.Sprintf("[curl-%s]", keyType))
	curlCmd.Stderr = logWriter(logView, fmt.Sprintf("[curl-%s]", keyType))
	err := curlCmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			appendLogs(logView, fmt.Sprintf("CURL command failed: %s\n", exitErr.Stderr))
		} else {
			appendLogs(logView, fmt.Sprintf("Error sending CURL request with %s: %v\n", keyType, err))
		}
	}
}

func logWriter(logView *tview.TextView, prefix string) *logWriterStruct {
	return &logWriterStruct{logView: logView, prefix: prefix}
}

type logWriterStruct struct {
	logView *tview.TextView
	prefix  string
}

func (w *logWriterStruct) Write(p []byte) (n int, err error) {
	appendLogs(w.logView, w.prefix+" "+string(p))
	return len(p), nil
}

func appendLogs(logView *tview.TextView, log string) {
	if logView != nil {
		logView.Write([]byte(log))
		logView.ScrollToEnd()
	}
}

func formatTTL(ttl time.Duration) string {
	if ttl < 0 {
		return "No Expiration"
	}
	return strconv.Itoa(int(ttl.Seconds()))
}

func countLines(view *tview.TextView) int {
	text := view.GetText(false)
	return len(strings.Split(text, "\n"))
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func clearLogs(logView *tview.TextView, lineCounter *tview.TextView) {
	logView.Clear()
	lineCounter.SetText("Line 0/0")
}
