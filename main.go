package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	version      = "1.0.0"
	defaultHost  = "http://localhost:11434"
	defaultModel = "qwen2.5-coder:14b-instruct"
)

const helixConfigSnippet = `
# ==============================================================================
# Helix Editor + Ollama AI Integration (hx-ollama)
# ==============================================================================

# Normal Mode Shortcuts (Space + o for Ollama)
[keys.normal.space.o]
g = "@:append-output<space>hx-ollama<space>generate<space>"
i = "@:insert-output<space>hx-ollama<space>generate<space>"
m = ":sh hx-ollama models"

# Visual / Selection Mode Shortcuts (Space + o for Ollama)
[keys.select.space.o]
e = "@|hx-ollama edit<space>"
f = ":pipe hx-ollama fix"
x = ":sh hx-ollama -f %val{filename} explain"
a = "@:sh<space>hx-ollama<space>-f<space>%val{filename}<space>ask<space>"
d = ":pipe hx-ollama docs"
c = ":pipe hx-ollama complete"
`

const systemPromptEdit = "You are an expert AI coding assistant integrated into the Helix text editor.\n" +
	"Your task is to edit, refactor, or rewrite the provided code based on the user's instructions.\n" +
	"CRITICAL RULE: Output ONLY the updated code. Do NOT wrap your output in markdown code blocks or ``` ``` fences.\n" +
	"Do NOT include any introduction, explanations, markdown formatting, or conversational text.\n" +
	"Your entire response will replace the user's selection in the editor."

const systemPromptFix = "You are an expert AI debugger integrated into the Helix text editor.\n" +
	"Your task is to analyze the provided code snippet, identify any syntax errors, logical bugs, or type mismatches, and fix them.\n" +
	"CRITICAL RULE: Output ONLY the corrected code. Do NOT wrap your output in markdown code blocks or ``` ``` fences.\n" +
	"Do NOT include any introduction, explanations, or conversational text.\n" +
	"Your entire response will replace the user's selection in the editor."

const systemPromptExplain = "You are an expert AI technical communicator writing a side-by-side code review scratchpad.\n" +
	"Analyze the provided code selection and explain clearly how it works.\n" +
	"Format your response as a clean, well-structured Markdown document using concise headings, bullet points, and code snippets as appropriate for the selection."

const systemPromptDocs = "You are an expert AI code documenter integrated into Helix text editor.\n" +
	"Add clear, concise docstrings, inline comments, and type hints/annotations to the provided code following standard style guidelines for the language.\n" +
	"CRITICAL RULE: Output ONLY the code with documentation added. Do NOT wrap your output in markdown code blocks or ``` ``` fences."

const systemPromptComplete = "You are an expert AI software developer integrated into the Helix text editor.\n" +
	"Your task is to complete missing code, logic, or function implementations in the provided selection.\n" +
	"CRITICAL RULE: Output ONLY the complete code. Do NOT wrap your output in markdown code blocks or ``` ``` fences.\n" +
	"Do NOT include any introduction, explanations, or conversational text."

const systemPromptAsk = "You are an expert AI technical assistant.\n" +
	"Analyze the provided code selection and answer the user's specific request or question clearly.\n" +
	"Use markdown headings, bullet points, and code snippets where helpful."

const systemPromptGenerate = "You are an expert AI software developer integrated into Helix text editor.\n" +
	"Generate clean, production-ready code based on the user's prompt instruction.\n" +
	"CRITICAL RULE: Output ONLY the generated code unless explicitly asked for explanation. Do NOT wrap your output in markdown code blocks or ``` ``` fences unless requested."

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
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
}

type TagsResponse struct {
	Models []ModelItem `json:"models"`
}

type ProjectConfig struct {
	CommentInstructions string `json:"_comment_instructions,omitempty"`
	Instructions        string `json:"instructions"`
	CommentModel        string `json:"_comment_model,omitempty"`
	Model               string `json:"model,omitempty"`
}

func loadProjectContext() (instructions string, modelOverride string) {
	dir, err := os.Getwd()
	if err != nil {
		return "", ""
	}

	for {
		jsonPath := filepath.Join(dir, ".hx-ollama.json")
		if data, err := os.ReadFile(jsonPath); err == nil {
			var pCfg ProjectConfig
			if err := json.Unmarshal(data, &pCfg); err == nil {
				return pCfg.Instructions, pCfg.Model
			}
		}

		txtPath := filepath.Join(dir, ".hx-ollama")
		if data, err := os.ReadFile(txtPath); err == nil {
			return strings.TrimSpace(string(data)), ""
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", ""
}

func handleContextCommand(args []string) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	cfgPath := filepath.Join(cwd, ".hx-ollama.json")
	instructions := ""
	if len(args) > 0 {
		instructions = strings.Join(args, " ")
	}

	pCfg := ProjectConfig{
		CommentInstructions: "Custom guidelines for this codebase (e.g. Python 3.11, FastAPI, C23, React + TS, etc.)",
		Instructions:        instructions,
		CommentModel:        "Optional model override for this specific project (leave empty to use global default)",
		Model:               "",
	}

	data, err := json.MarshalIndent(pCfg, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding .hx-ollama.json: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", cfgPath, err)
		os.Exit(1)
	}

	fmt.Printf("✅ Initialized project context file: %s\n", cfgPath)
	if instructions != "" {
		fmt.Printf("   Instructions set to: \"%s\"\n", instructions)
	} else {
		fmt.Println("   Edit .hx-ollama.json to add custom guidelines or model overrides for this repository.")
	}
}

func printHelp() {
	fmt.Printf("hx-ollama (v%s) — Ultra-Fast Local/LAN AI Integration for Helix Editor\n\n", version)
	fmt.Println("USAGE:")
	fmt.Println("  hx-ollama [OPTIONS] <ACTION> [PROMPT...]")
	fmt.Println("  echo \"code\" | hx-ollama [OPTIONS] <ACTION> [PROMPT...]")
	fmt.Println()
	fmt.Println("ACTIONS:")
	fmt.Println("  edit [prompt]     Refactor piped code according to prompt instruction")
	fmt.Println("  fix               Analyze and fix bugs, syntax, or logic errors in selection")
	fmt.Println("  explain           Explain selected code in detail (appends explanation below code)")
	fmt.Println("  docs              Add docstrings, comments, and type hints to selected code")
	fmt.Println("  complete          Complete missing code or logic in selection")
	fmt.Println("  generate <prompt> Generate new code from scratch for :append-output / :insert-output")
	fmt.Println("  models            List installed Ollama AI models on target host")
	fmt.Println("  setup / init      Display file locations and print Helix configuration snippet")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  -m, --model       Specify model tag (e.g. qwen2.5-coder:14b-instruct, deepseek-r1)")
	fmt.Println("  --host            Specify Ollama host URL (e.g. http://192.168.1.100:11434)")
	fmt.Println("  --raw             Force raw code output (strip code fences)")
	fmt.Println("  --markdown        Preserve markdown output (do not strip code fences)")
	fmt.Println("  --keep-code       Preserve original code selection above response")
	fmt.Println("  -v, --version     Show version number")
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
	fmt.Println("    :append-output hx-ollama generate \"write a python quicksort function\"")
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
		// Automatically generate a clean, commented configuration template
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

func normalizeHostURL(rawHost string) string {
	rawHost = strings.TrimSpace(rawHost)
	if rawHost == "" {
		return defaultHost
	}

	if !strings.HasPrefix(rawHost, "http://") && !strings.HasPrefix(rawHost, "https://") {
		rawHost = "http://" + rawHost
	}

	u, err := url.Parse(rawHost)
	if err != nil {
		return rawHost
	}

	if u.Port() == "" {
		// Add default Ollama port 11434 if no port was specified
		u.Host = net.JoinHostPort(u.Hostname(), "11434")
	}

	return strings.TrimRight(u.String(), "/")
}

func stripCodeFences(text string) string {
	s := strings.TrimSpace(text)
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) > 1 {
			lines = lines[1:]
			if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
				lines = lines[:len(lines)-1]
			}
			return strings.Join(lines, "\n")
		}
	}
	return text
}

func newHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   300 * time.Second, // Allow deep-reasoning models like deepseek-r1 sufficient time
	}
}

func main() {
	cfg := loadConfig()

	if envHost := os.Getenv("OLLAMA_HOST"); envHost != "" {
		cfg.Host = envHost
	}

	var (
		flagHost     string
		flagModel    string
		flagFile     string
		flagRaw      bool
		flagMarkdown bool
		flagKeepCode bool
		flagHelp     bool
		flagVersion  bool
	)

	flag.StringVar(&flagHost, "host", "", "Specify Ollama host URL")
	flag.StringVar(&flagModel, "m", "", "Specify model name")
	flag.StringVar(&flagModel, "model", "", "Specify model name")
	flag.StringVar(&flagFile, "f", "", "Specify file path to read code context from")
	flag.StringVar(&flagFile, "file", "", "Specify file path to read code context from")
	flag.BoolVar(&flagRaw, "raw", false, "Force raw code output")
	flag.BoolVar(&flagMarkdown, "markdown", false, "Preserve markdown output")
	flag.BoolVar(&flagKeepCode, "keep-code", false, "Preserve original code selection")
	flag.BoolVar(&flagVersion, "v", false, "Show version")
	flag.BoolVar(&flagVersion, "version", false, "Show version")
	flag.BoolVar(&flagHelp, "h", false, "Show help screen")
	flag.BoolVar(&flagHelp, "help", false, "Show help screen")

	flag.Usage = printHelp
	flag.Parse()

	if flagVersion {
		fmt.Printf("hx-ollama version %s\n", version)
		return
	}

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

	cfg.Host = normalizeHostURL(cfg.Host)

	if action == "context" || action == "init-project" {
		handleContextCommand(args[1:])
		return
	}

	projInstructions, projModel := loadProjectContext()
	if projModel != "" && flagModel == "" {
		cfg.Model = projModel
	}

	if action == "setup" || action == "init" || action == "install-helix" {
		fmt.Println("=================================================================")
		fmt.Printf("   hx-ollama (v%s) Go Static Binary Location Overview\n", version)
		fmt.Println("=================================================================")
		fmt.Println("1. Target Binary: ~/.local/bin/hx-ollama")
		fmt.Println("2. Config File:   ~/.config/hx-ollama/config.json")
		fmt.Println("3. Helix Config:  ~/.config/helix/config.toml")
		fmt.Println("=================================================================")
		fmt.Println("\nHelix Configuration Snippet:")
		fmt.Println(helixConfigSnippet)
		return
	}

	client := newHTTPClient()

	if action == "models" {
		url := cfg.Host + "/api/tags"
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
	if stdinText == "" && flagFile != "" {
		if data, err := os.ReadFile(flagFile); err == nil {
			stdinText = string(data)
			flagKeepCode = false
		}
	}
	sysPrompt := systemPromptEdit
	codeOnly := true

	switch action {
	case "fix":
		sysPrompt = systemPromptFix
	case "explain":
		sysPrompt = systemPromptExplain
		codeOnly = false
		flagKeepCode = true
	case "ask":
		sysPrompt = systemPromptAsk
		codeOnly = false
		flagKeepCode = true
	case "docs":
		sysPrompt = systemPromptDocs
	case "complete":
		sysPrompt = systemPromptComplete
	case "generate":
		sysPrompt = systemPromptGenerate
	}

	if projInstructions != "" {
		sysPrompt = fmt.Sprintf("Project Guidelines:\n%s\n\n%s", projInstructions, sysPrompt)
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

	url := cfg.Host + "/api/generate"
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
