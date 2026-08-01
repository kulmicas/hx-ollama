# `hx-ollama` 🦙⚡

> **Fast, Zero-Dependency Local & LAN AI Integration for Helix Editor**

> [!WARNING]
> ⚠️ **WARNING: THIS IS PURE AI SLOP!** 🤖✨  
> I made this to try out **vibecoding** with Google's **Antigravity**.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=for-the-badge)](LICENSE)
[![Platform: macOS | Linux](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Arch-darkgreen?style=for-the-badge)](https://github.com)
[![Editor: Helix](https://img.shields.io/badge/Editor-Helix-purple?style=for-the-badge&logo=helix)](https://helix-editor.com)

`hx-ollama` brings fast, private, local, and LAN-hosted LLM intelligence directly into the **Helix Editor (`hx`)**. It communicates with Ollama servers over HTTP REST and integrates seamlessly with Helix's native `:pipe` (`|`), `:append-output`, and `:insert-output` features.

---

## 📸 Overview

```text
Helix Editor  ──( visual selection )──>  hx-ollama  ──( HTTP REST )──>  Ollama (Local / LAN)
     │                                       │                                │
     └──────( in-place code replacement )────┴──────( clean code output )────┘
```

---

## ✨ Features

- ⚡ **Ultra-Fast & Self-Contained**: Compiles into a single, statically linked binary (`CGO_ENABLED=0`) with zero external runtime dependencies.
- 🌐 **Local & LAN AI Support**: Connects to Ollama running locally or on any remote server/machine on your network (`http://192.168.x.x:11434` or via `OLLAMA_HOST`).
- 🧼 **Smart Code Fence Stripping**: Automatically strips markdown fence wrappers (```python ... ```) for clean, drop-in code replacements in your buffer.
- 💡 **Preserves Code on Explanation**: The `explain` command appends structured markdown explanations *below* your code selection without overwriting your source code.
- 🛡️ **Fail-Safe Protection**: If your Ollama server is offline or unreachable, `hx-ollama` echoes your original code selection back to Helix so your highlighted code is **never deleted or lost**.
- ⚙️ **Simple JSON Configuration**: Easily set your default endpoint, model, and sampling temperature in `~/.config/hx-ollama/config.json`.

---

## ⚡ Quick Start

### 1. Build & Install (1 Command)

```bash
make install
```
*(This compiles `main.go` and installs the binary to `~/.local/bin/hx-ollama`).*

Or install directly with `go`:
```bash
go build -o ~/.local/bin/hx-ollama main.go
```

Make sure `~/.local/bin` is in your `PATH`.

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

## ⌨️ Helix Editor Configuration

Add the following keybindings to your Helix configuration file (`~/.config/helix/config.toml`):

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
x = "@y:vnew<ret>p:pipe hx-ollama explain<ret>:set-language markdown<ret>"
a = "@y:vnew<ret>p|hx-ollama ask<space>"
d = ":pipe hx-ollama docs"
c = ":pipe hx-ollama complete"
```

> **Note on Custom Questions (`Space + o + a`) & Explain Scratchpad (`Space + o + x`)**:
> - `Space + o + a`: Opens a Markdown scratchpad split (`:vnew`), leaving your cursor blinking so you can **type any custom question** about your selection!
> - `Space + o + x`: Streams a structured code breakdown into the scratchpad split.  
> Your original source file remains **100% untouched** on the left.

---

## 🛠️ Action Reference

| Action | Description | Keybinding / Command | Output Format |
| :--- | :--- | :--- | :--- |
| `edit [prompt]` | Refactors selection based on prompt instruction | `Space + o + e` | Raw Code |
| `fix` | Analyzes selection and fixes bugs or syntax errors | `Space + o + f` | Raw Code |
| `explain` | Explains selected code in side-by-side split | `Space + o + x` | Markdown Scratchpad Split |
| `ask [question]` | Answers any custom question about selection in split | `Space + o + a` | Markdown Scratchpad Split |
| `docs` | Adds docstrings, comments, and type hints to selection | `Space + o + d` | Raw Code |
| `complete` | Fills in missing functions or logic implementations | `Space + o + c` | Raw Code |
| `generate <prompt>` | Generates new code from scratch | `Space + o + g` | Raw Code |
| `models` | Lists installed Ollama models on host | `Space + o + m` | Terminal List |
| `context [text]` | Creates `.hx-ollama.json` project rules file | Terminal (`hx-ollama context`) | Local Config File |
| `setup` | Shows file locations and prints Helix config snippet | Terminal (`hx-ollama setup`) | Overview |

---

## 📁 Project-Local Context & Rules (`.hx-ollama.json`)

To set custom instructions or model overrides for a specific codebase, run in your terminal from inside your project folder:

```bash
hx-ollama context "Python 3.11 FastAPI project. Use PEP8 and native type hints."
```

This creates `.hx-ollama.json` in your project root:

```json
{
  "_comment_instructions": "Custom guidelines for this codebase (e.g. Python 3.11, FastAPI, C23, React + TS, etc.)",
  "instructions": "Python 3.11 FastAPI project. Use PEP8 and native type hints.",

  "_comment_model": "Optional model override for this specific project (leave empty to use global default)",
  "model": ""
}
```

`hx-ollama` automatically searches upwards from any subfolder to find `.hx-ollama.json` and include your project guidelines in all AI prompts.

---

## ⚙️ Configuration (`~/.config/hx-ollama/config.json`)

Running `make install` or `hx-ollama setup` automatically creates a commented template config file at `~/.config/hx-ollama/config.json`:

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

### Configuration Precedence

1. **CLI Flags** (`--host`, `-m`) *(Highest priority)*
2. **Environment Variable** (`OLLAMA_HOST`)
3. **Config File** (`~/.config/hx-ollama/config.json`)
4. **Built-in Defaults** (`http://localhost:11434`, `qwen2.5-coder:14b-instruct`)

---

## 🌐 Connecting to a Remote LAN Ollama Server

To connect `hx-ollama` to an Ollama instance running on another computer on your local network:

1. **Configure Ollama on the server machine to accept LAN connections**:
   - **macOS**: `launchctl setenv OLLAMA_HOST "0.0.0.0"`
   - **Linux**: Set `Environment="OLLAMA_HOST=0.0.0.0"` in `/etc/systemd/system/ollama.service` and restart.
   - **Windows**: Add user environment variable `OLLAMA_HOST` = `0.0.0.0`.
2. **Update your client config** (`~/.config/hx-ollama/config.json`):
   ```json
   {
     "host": "http://192.168.1.100:11434",
     "model": "qwen2.5-coder:14b-instruct"
   }
   ```
3. **Verify connection**:
   ```bash
   hx-ollama models
   ```

---

## 🚀 Terminal Help & Version

```bash
hx-ollama --help
hx-ollama --version
```

---

## 📄 License

Distributed under the [MIT License](LICENSE).
