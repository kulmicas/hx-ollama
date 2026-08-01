# `hx-ollama`: Portable Go Local/LAN AI Integration for Helix Editor

`hx-ollama` is a **fast, zero-dependency static binary** written in pure Go. It connects local or LAN-hosted **Ollama AI models** with **Helix Editor (`hx`)** via Helix's native `:pipe` (`|`), `:append-output`, and `:insert-output` features.

---

## ✨ Key Highlights

- ⚡ **Ultra-Fast & Self-Contained**: Statically linked binary (`CGO_ENABLED=0`), zero runtime dependencies.
- 🌐 **Local & LAN AI Support**: Easily connects to Ollama running locally or on another machine on your network (`http://192.168.x.x:11434` or `OLLAMA_HOST`).
- 🧼 **Smart Code Fence Stripping**: Automatically removes markdown code fence wrappers (```python ... ```) for clean in-place buffer replacement.
- 💡 **Preserves Code on Explanation**: `explain` appends structured explanations below your code without wiping it out.
- 🛡️ **Fail-Safe Fallback**: If Ollama is offline or hits an error, original code selection is preserved so Helix **never deletes your code**.

---

<details>
<summary><b>📖 Click to expand: Step-by-Step Guide to setting up Qwen 2.5 Coder (14B) in Ollama</b></summary>

<br>

### 1. Install Ollama
Download and install Ollama for macOS or Linux:
- **macOS**: `brew install ollama` (or download from [ollama.com](https://ollama.com))
- **Linux / Arch**: `curl -fsSL https://ollama.com/install.sh | sh`

### 2. Start the Ollama Service
```bash
ollama serve
```

### 3. Pull Qwen 2.5 Coder (14B Instruct)
Download the 14-billion parameter coding model:
```bash
ollama pull qwen2.5-coder:14b-instruct
```

### 4. Test the Model in Terminal
```bash
ollama run qwen2.5-coder:14b-instruct "write a python quicksort function"
```

### 5. Connect with `hx-ollama`
`hx-ollama` automatically uses `qwen2.5-coder:14b-instruct` by default! You can also set it explicitly in `~/.config/hx-ollama/config.json`.

</details>

---

## ⚡ Building & Installing

### 1. Build & Install (1 Command)
```bash
make install
```
*(This compiles `main.go` and installs the binary to `~/.local/bin/hx-ollama`).*

Or compile directly with `go`:
```bash
go build -o ~/.local/bin/hx-ollama main.go
```

---

## ⌨️ Helix Editor Configuration

Add the following keybindings to your Helix configuration (`~/.config/helix/config.toml`):

```toml
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
x = ":pipe hx-ollama explain"
d = ":pipe hx-ollama docs"
c = ":pipe hx-ollama complete"
```

---

## ⚙️ Configuration (`~/.config/hx-ollama/config.json`)

Running `hx-ollama setup` or `make install` creates a template config file at `~/.config/hx-ollama/config.json`:

```json
{
  "_comment_host": "URL of local or LAN Ollama server. Examples: http://localhost:11434 or http://192.168.1.100:11434",
  "host": "http://localhost:11434",

  "_comment_model": "Ollama model tag for coding (e.g. qwen2.5-coder:14b-instruct, deepseek-r1, codellama)",
  "model": "qwen2.5-coder:14b-instruct",

  "_comment_temperature": "Sampling temperature from 0.0 (precise code refactoring) to 1.0 (creative generation)",
  "temperature": 0.2
}
```

---

## 🚀 Terminal Usage & Help

```bash
hx-ollama --help
hx-ollama models
hx-ollama setup
```
