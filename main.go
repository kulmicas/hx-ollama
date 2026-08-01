package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultHost  = "http://localhost:11434"
	defaultModel = "qwen2.5-coder:14b-instruct"
)

const helixConfigSnippet = `
# ==============================================================================
# Helix Editor + Ollama AI Integration (hx-ollama)
# ==============================================================================

[keys.normal.space.o]
g = "@:append-output<space>hx-ollama<space>generate<space>"
i = "@:insert-output<space>hx-ollama<space>generate<space>"
m = ":sh hx-ollama models"

[keys.select.space.o]
e = "@|hx-ollama edit<space>"
f = ":pipe hx-ollama fix"
x = ":pipe hx-ollama explain"
d = ":pipe hx-ollama docs"
c = ":pipe hx-ollama complete"
`

const systemPromptEdit = `You are an expert AI coding assistant integrated into the Helix text editor.
Your task is to edit, refactor, or rewrite the provided code based on the user's instructions.
CRITICAL RULE: Output ONLY the updated code. Do NOT wrap your output in markdown code blocks or ``` ``` fences.
Do NOT include any introduction, explanations, markdown formatting, or conversational text.
Your entire response will replace the user's selection in the editor.`

const systemPromptFix = `You are an expert AI debugger integrated into the Helix text editor.
Your task is to analyze the provided code snippet, identify any syntax errors, logical bugs, or type mismatches, and fix them.
CRITICAL RULE: Output ONLY the corrected code. Do NOT wrap your output in markdown code blocks or ``` ``` fences.
Do NOT include any introduction, explanations, or conversational text.
Your entire response will replace the user's selection in the editor.`

const systemPromptExplain = `You are an expert software developer and technical communicator integrated into Helix text editor.
Analyze the provided code selection and explain clearly how it works, key data structures, algorithms, and potential edge cases.
Format your output with clear, concise markdown headings and bullet points.`

const systemPromptDocs = `You are an expert AI code documenter integrated into Helix text editor.
Add clear, concise docstrings, inline comments, and type hints/annotations to the provided code following standard style guidelines for the language.
CRITICAL RULE: Output ONLY the code with documentation added. Do NOT wrap your output in markdown code blocks or ``` ``` fences.`

const systemPromptGenerate = `You are an expert AI software developer integrated into Helix text editor.
Generate clean, production-ready code based on the user's prompt instruction.
CRITICAL RULE: Output ONLY the generated code unless explicitly asked for explanation. Do NOT wrap your output in markdown code blocks or ``` ``` fences unless requested.`

type Config struct {
	CommentHost        string  `json:"_comment_host,omitempty"`
	Host               string  `json:"host"`
	CommentModel       string  `json:"_comment_model,omitempty"`
	Model              string  `json:"model"`
	CommentTemperature string  `json:"_comment_temperature,omitempty"`
	Temperature        float64 `json:"temperature"`
}

type OllamaRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	System  string                 `json:"system,omitempty"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

type OllamaResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
	Done     bool   `json:"done"`
}

type ModelItem struct {
	Name string `json:"name"`
}

type TagsResponse struct {
	Models []ModelItem `json:"models"`
}

func printHelp() {
	fmt.Println("hx-ollama: Portable Go Static Binary for Helix Editor + Ollama AI")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  hx-ollama [OPTIONS] <ACTION> [PROMPT...]")
	fmt.Println("  echo \"code\" | hx-ollama [OPTIONS] <ACTION> [PROMPT...]")
	fmt.Println()
	fmt.Println("ACTIONS:")
	fmt.Println("  edit [prompt]     Refactor piped code according to prompt instruction")
	fmt.Println("  fix               Analyze and fix bugs, syntax, or logic errors in selection")
	fmt.Println("  explain           Explain selected code in detail (appends explanation below code)")
	fmt.Println("  docs              Add docstrings, comments, and type hints to selected code")
	fmt.Println("  complete          Complete missing logic in code selection")
	fmt.Println("  generate <prompt> Generate new code from scratch for :append-output / :insert-output")
	fmt.Println("  models            List installed Ollama AI models on host")
	fmt.Println("  setup / init      Display file locations and print Helix configuration snippet")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  -m <model>        Specify model (e.g. qwen2.5-coder:14b-instruct, deepseek-r1)")
	fmt.Println("  --host <url>      Specify Ollama host URL (e.g. http://192.168.1.100:11434)")
	fmt.Println("  --raw             Force raw code output (strip code fences)")
	fmt.Println("  --markdown        Preserve markdown output (do not strip code fences)")
	fmt.Println("  --keep-code       Preserve original code selection above response")
	fmt.Println("  -h, --help        Show this help screen")
	fmt.Println()
	fmt.Println("ENVIRONMENT VARIABLES:")
	fmt.Println("  OLLAMA_HOST       Default Ollama host URL (e.g. http://192.168.1.100:11434)")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  In Helix Visual Mode:")
	fmt.Println("    :pipe hx-ollama edit \"convert to async\"")
	fmt.Println("    :pipe hx-ollama fix")
	fmt.Println("    :pipe hx-ollama explain")
	fmt.Println()
	fmt.Println("  In Helix Normal Mode:")
	fmt.Println("    :append-output hx-ollama generate \"write a fibonacci function in python\"")
	fmt.Println()
	fmt.Println("  In Terminal:")
	fmt.Println("    hx-ollama models")
	fmt.Println("    hx-ollama setup")
}

func loadConfig() Config {
	cfg := Config{
		Host:        defaultHost,
		Model:       defaultModel,
		Temperature: 0.2,
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}

	cfgDir := filepath.Join(homeDir, ".config", "hx-ollama")
	cfgPath := filepath.Join(cfgDir, "config.json")

	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		return cfg
	}

	data, err := os.ReadFile(cfgPath)
	if os.IsNotExist(err) {
		// Create default commented template config file
		defaultCfg := Config{
			CommentHost:        "URL of local or LAN Ollama server. Examples: http://localhost:11434 or http://192.168.1.100:11434",
			Host:               defaultHost,
			CommentModel:       "Ollama model tag for coding (e.g. qwen2.5-coder:14b-instruct, deepseek-r1, codellama)",
			Model:              defaultModel,
			CommentTemperature: "Sampling temperature from 0.0 (precise code refactoring) to 1.0 (creative generation)",
			Temperature:        0.2,
		}
		if formatted, err := json.MarshalIndent(defaultCfg, "", "  "); err == nil {
			_ = os.WriteFile(cfgPath, formatted, 0644)
		}
		return cfg
	}

	if err == nil {
		var loaded Config
		if err := json.Unmarshal(data, &loaded); err == nil {
			if loaded.Host != "" {
				cfg.Host = loaded.Host
			}
			if loaded.Model != "" {
				cfg.Model = loaded.Model
			}
			if loaded.Temperature > 0 {
				cfg.Temperature = loaded.Temperature
			}
		}
	}

	return cfg
}

func stripCodeFences(text string) string {
	s := strings.TrimSpace(text)
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) > 1 {
			// Remove first line ```lang
			lines = lines[1:]
			// Remove last line ```
			if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
				lines = lines[:len(lines)-1]
			}
			return strings.Join(lines, "\n")
		}
	}
	return text
}

func main() {
	cfg := loadConfig()

	if envHost := os.Getenv("OLLAMA_HOST"); envHost != "" {
		cfg.Host = envHost
	}

	var (
		flagHost     string
		flagModel    string
		flagRaw      bool
		flagMarkdown bool
		flagKeepCode bool
		flagHelp     bool
	)

	flag.StringVar(&flagHost, "host", "", "Specify Ollama host URL")
	flag.StringVar(&flagModel, "m", "", "Specify model name")
	flag.BoolVar(&flagRaw, "raw", false, "Force raw code output")
	flag.BoolVar(&flagMarkdown, "markdown", false, "Preserve markdown output")
	flag.BoolVar(&flagKeepCode, "keep-code", false, "Preserve original code selection")
	flag.BoolVar(&flagHelp, "h", false, "Show help screen")
	flag.BoolVar(&flagHelp, "help", false, "Show help screen")

	flag.Usage = printHelp
	flag.Parse()

	args := flag.Args()

	if flagHelp || (len(args) == 0 && isTerminal(os.Stdin)) {
		printHelp()
		return
	}

	action := ""
	customPrompt := ""
	if len(args) > 0 {
		action = args[0]
	}
	if len(args) > 1 {
		customPrompt = strings.Join(args[1:], " ")
	}

	if action == "help" || action == "--help" || action == "-h" {
		printHelp()
		return
	}

	if flagHost != "" {
		cfg.Host = flagHost
	}
	if flagModel != "" {
		cfg.Model = flagModel
	}

	// Ensure Host starts with scheme http:// or https://
	if !strings.HasPrefix(cfg.Host, "http://") && !strings.HasPrefix(cfg.Host, "https://") {
		cfg.Host = "http://" + cfg.Host
	}

	if action == "setup" || action == "init" || action == "install-helix" {
		fmt.Println("=================================================================")
		fmt.Println("   hx-ollama Go Static Binary Location Overview")
		fmt.Println("=================================================================")
		fmt.Println("1. Target Binary: ~/.local/bin/hx-ollama")
		fmt.Println("2. Config File:   ~/.config/hx-ollama/config.json")
		fmt.Println("3. Helix Config:  ~/.config/helix/config.toml")
		fmt.Println("=================================================================")
		fmt.Println("\nHelix Configuration Snippet:")
		fmt.Println(helixConfigSnippet)
		return
	}

	client := &http.Client{Timeout: 120 * time.Second}

	if action == "models" {
		url := strings.TrimRight(cfg.Host, "/") + "/api/tags"
		resp, err := client.Get(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[hx-ollama Error]: Could not connect to Ollama at %s: %v\n", cfg.Host, err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var tags TagsResponse
		if err := json.Unmarshal(body, &tags); err == nil && len(tags.Models) > 0 {
			fmt.Printf("Installed Models on %s:\n", cfg.Host)
			for _, m := range tags.Models {
				fmt.Printf("  - %s\n", m.Name)
			}
		} else {
			fmt.Printf("No models found on %s or error parsing tags.\n", cfg.Host)
		}
		return
	}

	stdinText := readStdin()
	sysPrompt := systemPromptEdit
	codeOnly := true

	switch action {
	case "fix":
		sysPrompt = systemPromptFix
	case "explain":
		sysPrompt = systemPromptExplain
		codeOnly = false
		flagKeepCode = true
	case "docs":
		sysPrompt = systemPromptDocs
	case "generate":
		sysPrompt = systemPromptGenerate
	}

	if flagRaw {
		codeOnly = true
	}
	if flagMarkdown {
		codeOnly = false
	}

	fullPrompt := ""
	if stdinText != "" {
		pText := action
		if customPrompt != "" {
			pText = customPrompt
		}
		fullPrompt = fmt.Sprintf("User Request: %s\n\nCode Context:\n%s", pText, stdinText)
	} else {
		fullPrompt = fmt.Sprintf("%s %s", action, customPrompt)
	}

	reqBody := OllamaRequest{
		Model:  cfg.Model,
		Prompt: fullPrompt,
		System: sysPrompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": cfg.Temperature,
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		handleError(stdinText, fmt.Sprintf("Error encoding request JSON: %v", err), cfg)
		os.Exit(1)
	}

	url := strings.TrimRight(cfg.Host, "/") + "/api/generate"
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		handleError(stdinText, fmt.Sprintf("Could not connect to Ollama server at %s. Ensure 'ollama serve' is running.", cfg.Host), cfg)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		handleError(stdinText, fmt.Sprintf("Error reading response from Ollama: %v", err), cfg)
		os.Exit(1)
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(respBytes, &ollamaResp); err != nil {
		handleError(stdinText, fmt.Sprintf("Error parsing Ollama JSON response: %v", err), cfg)
		os.Exit(1)
	}

	if ollamaResp.Error != "" {
		handleError(stdinText, fmt.Sprintf("Ollama API Error: %s", ollamaResp.Error), cfg)
		os.Exit(1)
	}

	if ollamaResp.Response == "" {
		handleError(stdinText, "Received empty response from Ollama model.", cfg)
		os.Exit(1)
	}

	formatted := ollamaResp.Response
	if codeOnly {
		formatted = stripCodeFences(formatted)
	}

	if flagKeepCode && stdinText != "" {
		fmt.Printf("%s\n\n---\n### 💡 Code Explanation\n%s\n", stdinText, formatted)
	} else {
		fmt.Print(formatted)
	}
}

// Read stdin with 50ms timeout if non-blocking
func readStdin() string {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return ""
	}
	if (fi.Mode() & os.ModeCharDevice) != 0 {
		return ""
	}

	buf, err := io.ReadAll(os.Stdin)
	if err != nil {
		return ""
	}
	return string(buf)
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// Fail-safe fallback: Output original code back to stdout so Helix never deletes selected code on error!
func handleError(stdinText string, errMessage string, cfg Config) {
	if stdinText != "" {
		fmt.Print(stdinText)
	}
	fmt.Fprintf(os.Stderr, "\n[hx-ollama Error]: %s (Host: %s, Model: %s)\n", errMessage, cfg.Host, cfg.Model)
}
