// dev/llmbench benchmarks different strategies for calling the claude CLI.
//
// Usage:
//
//	go run ./dev/llmbench [--calls N] [--scenario ...] [--model ...]
//
// Scenarios: baseline, parallel, interactive, models, api
//
// Each scenario sends N identical trivial prompts and measures wall-clock time.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultPrompt = `Reply with ONLY the JSON object {"ok":true} — no other text.`

type callResult struct {
	Index    int
	Duration time.Duration
	Output   string
	Err      error
}

type scenarioResult struct {
	Name      string
	Total     time.Duration
	Calls     []callResult
	Processes int
}

func main() {
	numCalls := flag.Int("calls", 3, "number of LLM calls per scenario")
	scenario := flag.String("scenario", "all", "comma-separated scenarios: all, baseline, parallel, interactive, models, api")
	model := flag.String("model", "haiku", "claude model to use (for non-models scenarios)")
	apiKey := flag.String("api-key", "", "Anthropic API key for direct API scenario (or set ANTHROPIC_API_KEY)")
	flag.Parse()

	if *apiKey == "" {
		*apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if *apiKey == "" {
		// Try extracting OAuth token from macOS Keychain.
		if token, err := getKeychainOAuthToken(); err == nil {
			*apiKey = token
			fmt.Println("(Using OAuth token from macOS Keychain)")
		}
	}

	allScenarios := []string{"baseline", "parallel", "interactive", "models"}
	if *apiKey != "" {
		allScenarios = append(allScenarios, "api")
	}

	var toRun []string
	if *scenario == "all" {
		toRun = allScenarios
	} else {
		toRun = strings.Split(*scenario, ",")
	}

	fmt.Printf("LLM Call Benchmark — %d calls per scenario, model=%s\n", *numCalls, *model)
	fmt.Println(strings.Repeat("=", 70))

	var results []scenarioResult
	for _, name := range toRun {
		name = strings.TrimSpace(name)
		fmt.Printf("\n>>> Running scenario: %s\n", name)

		var result scenarioResult
		switch name {
		case "baseline":
			result = runBaseline(*numCalls, *model)
		case "parallel":
			result = runParallel(*numCalls, *model)
		case "interactive":
			result = runInteractive(*numCalls, *model)
		case "models":
			result = runModelComparison()
		case "api":
			result = runDirectAPI(*numCalls, *apiKey)
		default:
			fmt.Fprintf(os.Stderr, "unknown scenario: %s\n", name)
			continue
		}
		results = append(results, result)
		printResult(result)

		// Brief pause between scenarios to avoid rate limiting.
		if len(toRun) > 1 {
			time.Sleep(2 * time.Second)
		}
	}

	if len(results) > 1 {
		fmt.Println()
		fmt.Println(strings.Repeat("=", 70))
		fmt.Println("SUMMARY")
		fmt.Println(strings.Repeat("=", 70))
		for _, r := range results {
			errs := 0
			for _, c := range r.Calls {
				if c.Err != nil {
					errs++
				}
			}
			avg := time.Duration(0)
			if len(r.Calls) > 0 {
				var sum time.Duration
				for _, c := range r.Calls {
					sum += c.Duration
				}
				avg = sum / time.Duration(len(r.Calls))
			}
			fmt.Printf("  %-20s  total=%-10s  avg=%-10s  procs=%-2d  errors=%d\n",
				r.Name, r.Total.Round(time.Millisecond), avg.Round(time.Millisecond), r.Processes, errs)
		}
	}
}

// runBaseline: N sequential claude --print calls (current behavior).
func runBaseline(n int, model string) scenarioResult {
	calls := make([]callResult, n)
	start := time.Now()

	for i := 0; i < n; i++ {
		t := time.Now()
		out, err := runClaude(context.Background(), model, defaultPrompt)
		calls[i] = callResult{Index: i, Duration: time.Since(t), Output: out, Err: err}
		fmt.Printf("  call %d: %s\n", i+1, calls[i].Duration.Round(time.Millisecond))
	}

	return scenarioResult{Name: "baseline", Total: time.Since(start), Calls: calls, Processes: n}
}

// runParallel: N concurrent claude --print calls.
func runParallel(n int, model string) scenarioResult {
	calls := make([]callResult, n)
	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			t := time.Now()
			out, err := runClaude(context.Background(), model, defaultPrompt)
			calls[idx] = callResult{Index: idx, Duration: time.Since(t), Output: out, Err: err}
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		fmt.Printf("  call %d: %s\n", i+1, calls[i].Duration.Round(time.Millisecond))
	}

	return scenarioResult{Name: "parallel", Total: time.Since(start), Calls: calls, Processes: n}
}

// runInteractive: one persistent claude process (not --print) with stream-json I/O.
// Uses interactive mode so the process stays alive between prompts.
func runInteractive(n int, model string) scenarioResult {
	calls := make([]callResult, n)
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Start claude with --print and stream-json I/O.
	// With --input-format stream-json, the process reads NDJSON from stdin
	// and should stay alive for multiple messages.
	cmd := exec.CommandContext(ctx, "claude",
		"--print",
		"--model", model,
		"--output-format", "stream-json",
		"--input-format", "stream-json",
		"--verbose",
		"--no-session-persistence",
		"--dangerously-skip-permissions",
		"--disable-slash-commands",
	)
	cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("  ERROR creating stdout pipe: %v\n", err)
		return scenarioResult{Name: "interactive", Total: time.Since(start), Calls: calls, Processes: 0}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Printf("  ERROR creating stdin pipe: %v\n", err)
		return scenarioResult{Name: "interactive", Total: time.Since(start), Calls: calls, Processes: 0}
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("  ERROR starting claude: %v\n", err)
		return scenarioResult{Name: "interactive", Total: time.Since(start), Calls: calls, Processes: 0}
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	// With stream-json input, we send the first message via stdin
	// (no -p flag needed — stdin IS the input).
	fmt.Printf("  process started, sending first prompt...\n")

	// Send N prompts via stdin, reading stream-json responses.
	for i := 0; i < n; i++ {
		t := time.Now()

		// Send user message as NDJSON.
		msg := map[string]any{
			"type": "user",
			"message": map[string]any{
				"role": "user",
				"content": defaultPrompt,
			},
		}
		msgBytes, _ := json.Marshal(msg)
		msgBytes = append(msgBytes, '\n')

		if _, werr := stdin.Write(msgBytes); werr != nil {
			calls[i] = callResult{Index: i, Duration: time.Since(t), Err: fmt.Errorf("stdin write: %w", werr)}
			fmt.Printf("  call %d: ERROR write %v\n", i+1, werr)
			break
		}

		out, rerr := readStreamResult(scanner)
		calls[i] = callResult{Index: i, Duration: time.Since(t), Output: out, Err: rerr}
		if rerr != nil {
			fmt.Printf("  call %d: %s  ERROR %v\n", i+1, calls[i].Duration.Round(time.Millisecond), rerr)
		} else {
			fmt.Printf("  call %d: %s\n", i+1, calls[i].Duration.Round(time.Millisecond))
		}
	}

	stdin.Close()
	cmd.Process.Kill()
	cmd.Wait()

	return scenarioResult{Name: "interactive", Total: time.Since(start), Calls: calls, Processes: 1}
}

// runModelComparison: one call per model to compare latency.
func runModelComparison() scenarioResult {
	models := []string{"haiku", "sonnet", "opus"}
	calls := make([]callResult, len(models))
	start := time.Now()

	for i, m := range models {
		t := time.Now()
		out, err := runClaude(context.Background(), m, defaultPrompt)
		calls[i] = callResult{Index: i, Duration: time.Since(t), Output: out, Err: err}
		if err != nil {
			fmt.Printf("  %-8s  %s  ERROR: %v\n", m, calls[i].Duration.Round(time.Millisecond), err)
		} else {
			fmt.Printf("  %-8s  %s  output=%q\n", m, calls[i].Duration.Round(time.Millisecond), truncate(out, 60))
		}
	}

	return scenarioResult{Name: "models", Total: time.Since(start), Calls: calls, Processes: len(models)}
}

// runDirectAPI: call the Anthropic Messages API directly with net/http.
func runDirectAPI(n int, apiKey string) scenarioResult {
	calls := make([]callResult, n)
	start := time.Now()

	for i := 0; i < n; i++ {
		t := time.Now()
		out, err := callAnthropicAPI(context.Background(), apiKey, defaultPrompt)
		calls[i] = callResult{Index: i, Duration: time.Since(t), Output: out, Err: err}
		if err != nil {
			fmt.Printf("  call %d: %s  ERROR: %v\n", i+1, calls[i].Duration.Round(time.Millisecond), err)
		} else {
			fmt.Printf("  call %d: %s\n", i+1, calls[i].Duration.Round(time.Millisecond))
		}
	}

	return scenarioResult{Name: "api-direct", Total: time.Since(start), Calls: calls, Processes: 0}
}

// callAnthropicAPI makes a direct HTTP call to the Anthropic Messages API.
func callAnthropicAPI(ctx context.Context, apiKey, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	body := map[string]any{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 64,
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	// OAuth tokens use Bearer auth; raw API keys use x-api-key.
	if strings.HasPrefix(apiKey, "sk-ant-oat") {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	} else {
		req.Header.Set("x-api-key", apiKey)
	}
	req.Header.Set("anthropic-version", "2023-06-01")
	// OAuth tokens require this beta header to be accepted on /v1/messages.
	if strings.HasPrefix(apiKey, "sk-ant-oat") {
		req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode API response: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("API error: %s", result.Error.Message)
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty API response (status %d)", resp.StatusCode)
	}

	return result.Content[0].Text, nil
}

// waitForReady reads stream-json lines until we see a system_prompt or init message,
// indicating the interactive session is ready for input.
func waitForReady(scanner *bufio.Scanner) error {
	timeout := time.After(120 * time.Second)
	ready := make(chan error, 1)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			var msg map[string]any
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				continue
			}
			msgType, _ := msg["type"].(string)
			// Interactive mode sends various init messages. We look for
			// something that indicates it's ready to receive user input.
			switch msgType {
			case "system", "system_prompt", "init", "ready":
				ready <- nil
				return
			}
		}
		ready <- fmt.Errorf("stream ended before ready signal")
	}()

	select {
	case err := <-ready:
		return err
	case <-timeout:
		return fmt.Errorf("timeout waiting for ready")
	}
}

// readStreamResult reads stream-json lines until we get a result message.
func readStreamResult(scanner *bufio.Scanner) (string, error) {
	timeout := time.After(90 * time.Second)
	type parseResult struct {
		content string
		err     error
	}
	done := make(chan parseResult, 1)

	go func() {
		var lastContent string
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var msg map[string]any
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				continue
			}

			msgType, _ := msg["type"].(string)

			switch msgType {
			case "assistant":
				if content, ok := msg["content"].(string); ok {
					lastContent = content
				}
				if blocks, ok := msg["content"].([]any); ok {
					for _, b := range blocks {
						if block, ok := b.(map[string]any); ok {
							if text, ok := block["text"].(string); ok {
								lastContent += text
							}
						}
					}
				}
			case "result":
				if content, ok := msg["result"].(string); ok {
					done <- parseResult{content: content}
					return
				}
				done <- parseResult{content: lastContent}
				return
			case "error":
				errMsg, _ := msg["error"].(string)
				done <- parseResult{err: fmt.Errorf("stream error: %s", errMsg)}
				return
			default:
				// Log unknown message types for debugging stream protocol.
				fmt.Fprintf(os.Stderr, "  [stream debug] type=%q\n", msgType)
			}
		}
		if lastContent != "" {
			done <- parseResult{content: lastContent}
		} else {
			done <- parseResult{err: fmt.Errorf("stream ended without result")}
		}
	}()

	select {
	case r := <-done:
		return r.content, r.err
	case <-timeout:
		return "", fmt.Errorf("timeout reading stream response")
	}
}

// runClaude executes a single claude --print call.
func runClaude(ctx context.Context, model, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude",
		"--print",
		"--model", model,
		"--no-session-persistence",
		"--dangerously-skip-permissions",
		"--disable-slash-commands",
		"-p", prompt,
	)
	cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude --print failed: %w\nstderr: %s", err, string(exitErr.Stderr))
		}
		return "", fmt.Errorf("claude --print failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// getKeychainOAuthToken extracts the Claude Code OAuth access token from macOS Keychain.
func getKeychainOAuthToken() (string, error) {
	out, err := exec.Command("security", "find-generic-password",
		"-s", "Claude Code-credentials",
		"-a", os.Getenv("USER"),
		"-w",
	).Output()
	if err != nil {
		return "", fmt.Errorf("keychain read failed: %w", err)
	}

	var creds struct {
		ClaudeAiOauth struct {
			AccessToken string `json:"accessToken"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(out, &creds); err != nil {
		return "", fmt.Errorf("failed to parse keychain credentials: %w", err)
	}
	if creds.ClaudeAiOauth.AccessToken == "" {
		return "", fmt.Errorf("no accessToken in keychain credentials")
	}
	return creds.ClaudeAiOauth.AccessToken, nil
}

func filterEnv(env []string, exclude string) []string {
	var filtered []string
	for _, e := range env {
		if !strings.HasPrefix(e, exclude+"=") {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func printResult(r scenarioResult) {
	if len(r.Calls) == 0 {
		fmt.Printf("\n  %-20s  no calls\n", r.Name)
		return
	}

	var durations []time.Duration
	errs := 0
	for _, c := range r.Calls {
		if c.Err != nil {
			errs++
		}
		durations = append(durations, c.Duration)
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	avg := sum / time.Duration(len(durations))

	fmt.Printf("\n  %-20s  total=%-10s  avg=%-10s  min=%-10s  max=%-10s  procs=%d  errors=%d\n",
		r.Name,
		r.Total.Round(time.Millisecond),
		avg.Round(time.Millisecond),
		durations[0].Round(time.Millisecond),
		durations[len(durations)-1].Round(time.Millisecond),
		r.Processes,
		errs,
	)
}
