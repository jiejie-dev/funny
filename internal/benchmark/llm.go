package benchmark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const llmSystemPrompt = `You write code in the funny programming language.
Reply with ONLY valid funny source code for the task — no markdown fences, no explanation.`

// Provider names for CLI and reports.
const (
	ProviderMock      = "mock"
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
)

// LLMOptions configures a remote guess generator.
type LLMOptions struct {
	Provider string
	Model    string
	APIKey   string
	Timeout  time.Duration
}

// NewGuessGenerator returns a generator for the given provider.
// mock echoes the prompt; openai/anthropic call HTTP APIs (APIKey required).
func NewGuessGenerator(opts LLMOptions) (GuessGenerator, error) {
	switch opts.Provider {
	case "", ProviderMock:
		return func(prompt string) (string, error) {
			return GenerateLLMGuess(prompt), nil
		}, nil
	case ProviderOpenAI:
		if opts.APIKey == "" {
			return nil, fmt.Errorf("openai: OPENAI_API_KEY is required")
		}
		model := opts.Model
		if model == "" {
			model = "gpt-4o-mini"
		}
		return openAIGenerator(opts.APIKey, model, opts.Timeout), nil
	case ProviderAnthropic:
		if opts.APIKey == "" {
			return nil, fmt.Errorf("anthropic: ANTHROPIC_API_KEY is required")
		}
		model := opts.Model
		if model == "" {
			model = "claude-3-5-haiku-20241022"
		}
		return anthropicGenerator(opts.APIKey, model, opts.Timeout), nil
	default:
		return nil, fmt.Errorf("unknown provider %q (use mock, openai, anthropic)", opts.Provider)
	}
}

// GeneratorFromEnv builds a generator using provider flags and environment keys.
func GeneratorFromEnv(provider, model string) (GuessGenerator, string, string, error) {
	opts := LLMOptions{
		Provider: provider,
		Model:    model,
		Timeout:  120 * time.Second,
	}
	switch provider {
	case "", ProviderMock:
		opts.Provider = ProviderMock
	case ProviderOpenAI:
		opts.APIKey = os.Getenv("OPENAI_API_KEY")
	case ProviderAnthropic:
		opts.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	default:
		return nil, "", "", fmt.Errorf("unknown provider %q", provider)
	}
	gen, err := NewGuessGenerator(opts)
	return gen, opts.Provider, opts.Model, err
}

func openAIGenerator(apiKey, model string, timeout time.Duration) GuessGenerator {
	client := &http.Client{Timeout: timeout}
	return func(prompt string) (string, error) {
		body := map[string]any{
			"model": model,
			"messages": []map[string]string{
				{"role": "system", "content": llmSystemPrompt},
				{"role": "user", "content": prompt},
			},
			"temperature": 0,
		}
		raw, err := json.Marshal(body)
		if err != nil {
			return "", err
		}
		req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(raw))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("openai HTTP %d: %s", resp.StatusCode, truncate(string(data), 400))
		}
		var parsed struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(data, &parsed); err != nil {
			return "", err
		}
		if len(parsed.Choices) == 0 {
			return "", fmt.Errorf("openai: empty choices")
		}
		return stripCodeFences(parsed.Choices[0].Message.Content), nil
	}
}

func anthropicGenerator(apiKey, model string, timeout time.Duration) GuessGenerator {
	client := &http.Client{Timeout: timeout}
	return func(prompt string) (string, error) {
		body := map[string]any{
			"model":      model,
			"max_tokens": 1024,
			"system":     llmSystemPrompt,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
		}
		raw, err := json.Marshal(body)
		if err != nil {
			return "", err
		}
		req, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(raw))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("anthropic HTTP %d: %s", resp.StatusCode, truncate(string(data), 400))
		}
		var parsed struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.Unmarshal(data, &parsed); err != nil {
			return "", err
		}
		if len(parsed.Content) == 0 {
			return "", fmt.Errorf("anthropic: empty content")
		}
		return stripCodeFences(parsed.Content[0].Text), nil
	}
}

func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) >= 2 {
			end := len(lines)
			if strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
				end--
			}
			start := 1
			if len(lines) > 1 && strings.TrimSpace(lines[1]) != "" {
				// drop ```funny or ``` line
			}
			s = strings.Join(lines[start:end], "\n")
		}
	}
	return strings.TrimSpace(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// RunReport executes the benchmark and returns a JSON-serializable report.
func RunReport(tasks []Task, gen GuessGenerator, provider, model string) Report {
	results, passRate := RunWithGenerator(tasks, gen)
	return Report{
		Provider: provider,
		Model:    model,
		PassRate: passRate,
		Results:  results,
	}
}
