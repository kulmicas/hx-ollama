package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ==============================================================================
// 1. Configuration & LAN Endpoint Support
// ==============================================================================

type Config struct {
	Host            string   `json:"host"`
	Model           string   `json:"model"`
	Temperature     float64  `json:"temperature"`
	TimeoutSeconds  int      `json:"timeout_seconds"`
	PreferredModels []string `json:"preferred_models"`
}

var defaultConfig = Config{
	Host:           "http://localhost:11434",
	Model:          "",
	Temperature:    0.2,
	TimeoutSeconds: 60,
	PreferredModels: []string{
		"qwen2.5-coder",
		"qwen2.5-coder:14b-instruct",
		"qwen2.5-coder:7b",
		"qwen2.5-coder:1.5b",
		"deepseek-r1",
		"codellama",
		"llama3.2",
	},
}

func getConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".config/hx-ollama"
	}
	return filepath.Join(home, ".config", "hx-ollama")
}

func getConfigFile() string {
	return filepath.Join(getConfigDir(), "config.json")
}

func loadConfig() Config {
	cfg := defaultConfig
	path := getConfigFile()

	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, &cfg)
	}

	if envHost := os.Getenv("OLLAMA_HOST"); envHost != "" {
		cfg.Host = envHost
	}

	return cfg
}

func saveDefaultConfig() error {
	dir := getConfigDir()
	path := getConfigFile()

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil {
		return nil
	}

	sampleConfig := `{
  "host": "http://localhost:11434",
  "model": "",
  "temperature": 0.2,
  "timeout_seconds": 60,
  "preferred_models": [
    "qwen2.5-coder",
    "qwen2.5-coder:14b-instruct",
    "qwen2.5-coder:7b",
    "deepseek-r1",
    "codellama"
  ]
}`

	return os.WriteFile(path, []byte(sampleConfig), 0644)
}

// ==============================================================================
// 2. Ollama API Client
// ==============================================================================

type OllamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

type OllamaGenerateRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	System  string                 `json:"system,omitempty"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options"`
}

type OllamaGenerateResponse struct {
	Response string `json:"response"`
}

type OllamaClient struct {
	Host    string
	Timeout time.Duration
}

func NewOllamaClient(host string, timeoutSec int) *OllamaClient {
	host = strings.TrimRight(host, "/")
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}
	return &OllamaClient{
		Host:    host,
		Timeout: time.Duration(timeoutSec) * time.Second,
	}
}

func (c *OllamaClient) ListModels() ([]string, error) {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(c.Host + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("could not connect to Ollama at %s: %w", c.Host, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama returned HTTP %d", resp.StatusCode)
	}

	var tags OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}

	var names []string
	for _, m := range tags.Models {
		names = append(names, m.Name)
	}
	return names, nil
}

func (c *OllamaClient) ResolveModel(requested string, preferred []string) (string, error) {
	installed, err := c.ListModels()
	if err != nil || len(installed) == 0 {
		if requested != "" {
			return requested, nil
		}
		return "", fmt.Errorf("no models found on Ollama server at %s. Please run 'ollama pull qwen2.5-coder'", c.Host)
	}

	if requested != "" {
		for _, m := range installed {
			if m == requested || strings.HasPrefix(m, requested+":") {
				return m, nil
			}
		}
		return requested, nil
	}

	for _, pref := range preferred {
		for _, inst := range installed {
			if inst == pref || strings.HasPrefix(inst, pref+":") || strings.Contains(inst, pref) {
				return inst, nil
			}
		}
	}

	return installed[0], nil
}

func (c *OllamaClient) Generate(model, prompt, system string, temp float64) (string, error) {
	client := http.Client{Timeout: c.Timeout}

	reqBody := OllamaGenerateRequest{
		Model:  model,
		Prompt: prompt,
		System: system,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": temp,
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := client.Post(c.Host+"/api/generate", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("failed to reach Ollama at %s: %w", c.Host, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama returned HTTP %d", resp.StatusCode)
	}

	var genResp OllamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", err
	}

	return genResp.Response, nil
}

// ==============================================================================
// 3. Formatter & Code Fence Stripping
// ==============================================================================

func stripCodeFences(text string) string {
	if text == "" {
		return text
	}

	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) >= 2 && strings.HasPrefix(lines[0], "```") && strings.TrimSpace(lines[len(lines)-1]) == "```" {
		lines = lines[1 : len(lines)-1]
		res := strings.Join(lines, "\n")
		if strings.HasSuffix(text, "\n") {
			res += "\n"
		}
		return res
	}

	re := regexp.MustCompile("(?s)```(?:\\w+)?\n(.*?)```")
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}

	return text
}

func formatOutput(raw string, codeOnly bool) string {
	if codeOnly {
		return stripCodeFences(raw)
	}
	return raw
}

// ==============================================================================
// 4. Stdin Reading & System Prompts
// ==============================================================================

func readStdinNonBlocking() string {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		buf, err := io.ReadAll(os.Stdin)
		if err == nil {
			return string(buf)
		}
	}
	return ""
}

const (
	systemPromptEdit = `You are an expert AI coding assistant integrated into the Helix text editor.
Your task is to edit, refactor, or rewrite the provided code based on the user's instructions.
CRITICAL RULE: Output ONLY the updated code. Do NOT wrap your output in markdown code blocks or ``` ``` fences.
Do NOT include any introduction, explanations, markdown formatting, or conversational text.
Your entire response will replace the user's selection in the editor.`

	systemPromptFix = `You are an expert AI debugger integrated into the Helix text editor.
Your task is to analyze the provided code snippet, identify any syntax errors, logical bugs, or type mismatches, and fix them.
CRITICAL RULE: Output ONLY the corrected code. Do NOT wrap your output in markdown code blocks or ``` ``` fences.
Do NOT include any introduction, explanations, or conversational text.
Your entire response will replace the user's selection in the editor.`

	systemPromptExplain = `You are an expert software developer and technical communicator integrated into Helix text editor.
Analyze the provided code selection and explain clearly how it works, key data structures, algorithms, and potential edge cases.
Format your output with clear, concise markdown headings and bullet points.`

	systemPromptDocs = `You are an expert AI code documenter integrated into Helix text editor.
Add clear, concise docstrings, inline comments, and type hints/annotations to the provided code following standard style guidelines for the language.
CRITICAL RULE: Output ONLY the code with documentation added. Do NOT wrap your output in markdown code blocks or ``` ``` fences.`

	systemPromptComplete = `You are an inline code completion AI integrated into Helix text editor.
Complete the code logic naturally following the context and existing patterns.
CRITICAL RULE: Output ONLY the completion code. Do NOT wrap your output in markdown code blocks or ``` ``` fences.`

	systemPromptGenerate = `You are an expert AI software developer integrated into Helix text editor.
Generate clean, production-ready code based on the user's prompt instruction.
CRITICAL RULE: Output ONLY the generated code unless explicitly asked for explanation. Do NOT wrap your output in markdown code blocks or ``` ``` fences unless requested.`
)

const helixConfigSnippet = `
# ==============================================================================
# Helix Editor + Ollama AI Integration (hx-ollama)
# ==============================================================================

# Normal Mode Keybindings (Space + o for Ollama)
[keys.normal.space.o]
g = "@:append-output<space>hx-ollama<space>generate<space>"
i = "@:insert-output<space>hx-ollama<space>generate<space>"
m = ":sh hx-ollama models"

# Visual / Selection Mode Keybindings (Space + o for Ollama)
[keys.select.space.o]
e = "@|hx-ollama edit<space>"
f = ":pipe hx-ollama fix"
x = ":pipe hx-ollama explain"
d = ":pipe hx-ollama docs"
c = ":pipe hx-ollama complete"
`

// ==============================================================================
// 5. Interactive Setup & Main Entrypoint
// ==============================================================================

func runInteractiveSetup() {
	home, _ := os.UserHomeDir()
	binPath := filepath.Join(home, ".local", "bin", "hx-ollama")
	cfgPath := getConfigFile()
	helixPath := filepath.Join(home, ".config", "helix", "config.toml")

	fmt.Println("=================================================================")
	fmt.Println("   hx-ollama Interactive Setup & Location Overview")
	fmt.Println("=================================================================")
	fmt.Printf("1. Executable Location:   %s\n", binPath)
	fmt.Printf("2. Config File Location:   %s\n", cfgPath)
	fmt.Printf("3. Helix Config Location:  %s\n", helixPath)
	fmt.Println("=================================================================")
	fmt.Println()

	_ = saveDefaultConfig()
	fmt.Printf("✅ Verified config file at %s\n", cfgPath)

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("❓ Do you want to append Space + o keybindings to %s? [y/N]: ", helixPath)
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(strings.ToLower(ans))

	if ans == "y" || ans == "yes" {
		os.MkdirAll(filepath.Dir(helixPath), 0755)
		existing, _ := os.ReadFile(helixPath)
		if strings.Contains(string(existing), "hx-ollama") {
			fmt.Printf("ℹ️  hx-ollama keybindings are already present in %s\n", helixPath)
		} else {
			f, err := os.OpenFile(helixPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
					f.WriteString("\n")
				}
				f.WriteString(helixConfigSnippet)
				f.Close()
				fmt.Printf("✅ Successfully added keybindings to %s!\n", helixPath)
			}
		}
	} else {
		fmt.Println("⏭️  Skipped Helix config update.")
	}

	fmt.Println()
	fmt.Println("🎉 Setup complete! Edit ~/.config/hx-ollama/config.json to change AI endpoint or models.")
}

func main() {
	cfg := loadConfig()

	modelFlag := flag.String("m", "", "Specify Ollama model")
	hostFlag := flag.String("host", "", "Ollama host URL (e.g. http://192.168.1.100:11434)")
	tempFlag := flag.Float64("t", -1.0, "Temperature setting")
	rawFlag := flag.Bool("raw", false, "Force raw code output")
	markdownFlag := flag.Bool("markdown", false, "Preserve markdown output")
	keepCodeFlag := flag.Bool("keep-code", false, "Preserve original code and append explanation")

	flag.Parse()

	args := flag.Args()
	action := ""
	extraPrompt := ""
	if len(args) > 0 {
		action = strings.ToLower(args[0])
		extraPrompt = strings.Join(args[1:], " ")
	}

	stdinContent := readStdinNonBlocking()

	if action == "setup" || action == "init" || action == "install-helix" {
		runInteractiveSetup()
		return
	}

	if action == "setup-helix" {
		fmt.Println(helixConfigSnippet)
		return
	}

	host := cfg.Host
	if *hostFlag != "" {
		host = *hostFlag
	}

	temp := cfg.Temperature
	if *tempFlag >= 0.0 {
		temp = *tempFlag
	}

	client := NewOllamaClient(host, cfg.TimeoutSeconds)

	if action == "models" {
		models, err := client.ListModels()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[hx-ollama] Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Installed Models on %s:\n", host)
		for _, m := range models {
			fmt.Printf("  - %s\n", m)
		}
		return
	}

	codeOnly := true
	keepCode := *keepCodeFlag
	systemPrompt := systemPromptEdit
	userPrompt := ""

	switch action {
	case "fix":
		systemPrompt = systemPromptFix
		if extraPrompt != "" {
			userPrompt = extraPrompt
		} else {
			userPrompt = "Fix any bugs, syntax errors, or logical issues in this code."
		}
	case "explain":
		systemPrompt = systemPromptExplain
		if extraPrompt != "" {
			userPrompt = extraPrompt
		} else {
			userPrompt = "Explain this code in detail."
		}
		codeOnly = false
		keepCode = true
	case "docs":
		systemPrompt = systemPromptDocs
		if extraPrompt != "" {
			userPrompt = extraPrompt
		} else {
			userPrompt = "Add clear docstrings and comments to this code."
		}
	case "complete":
		systemPrompt = systemPromptComplete
		if extraPrompt != "" {
			userPrompt = extraPrompt
		} else {
			userPrompt = "Complete the logic for this code snippet."
		}
	case "generate", "create":
		systemPrompt = systemPromptGenerate
		if extraPrompt != "" {
			userPrompt = extraPrompt
		} else {
			userPrompt = strings.Join(args, " ")
		}
	case "edit", "refactor", "change":
		systemPrompt = systemPromptEdit
		if extraPrompt != "" {
			userPrompt = extraPrompt
		} else {
			userPrompt = "Refactor and clean up this code."
		}
	default:
		fullInstruction := strings.Join(args, " ")
		if stdinContent != "" {
			systemPrompt = systemPromptEdit
			if fullInstruction != "" {
				userPrompt = fullInstruction
			} else {
				userPrompt = "Improve and refine this code."
			}
		} else {
			systemPrompt = systemPromptGenerate
			if fullInstruction != "" {
				userPrompt = fullInstruction
			} else {
				userPrompt = "Generate the requested code."
			}
		}
	}

	fullPrompt := ""
	if stdinContent != "" {
		fullPrompt = fmt.Sprintf("User Request: %s\n\nCode Context:\n%s", userPrompt, stdinContent)
	} else {
		fullPrompt = userPrompt
	}

	if strings.TrimSpace(fullPrompt) == "" {
		flag.Usage()
		os.Exit(1)
	}

	if *rawFlag {
		codeOnly = true
	} else if *markdownFlag {
		codeOnly = false
	}

	reqModel := *modelFlag
	if reqModel == "" {
		reqModel = cfg.Model
	}

	model, err := client.ResolveModel(reqModel, cfg.PreferredModels)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[hx-ollama] Error resolving model: %v\n", err)
		os.Exit(1)
	}

	rawResp, err := client.Generate(model, fullPrompt, systemPrompt, temp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[hx-ollama] Error: %v\n", err)
		os.Exit(1)
	}

	formatted := formatOutput(rawResp, codeOnly)
	if keepCode && stdinContent != "" {
		formatted = fmt.Sprintf("%s\n\n---\n### 💡 Code Explanation\n%s\n", strings.TrimRight(stdinContent, "\r\n"), formatted)
	}

	fmt.Print(formatted)
}
