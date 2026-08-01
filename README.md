# `hx-ollama`: Helix Editor + Ollama AI Integration

`hx-ollama` is a fast, lightweight, zero-dependency tool designed to seamlessly connect local AI models running on **Ollama** with **Helix Editor** (`hx`) via Helix's native `:pipe` (`|`), `:append-output`, and `:insert-output` features.

---

## ✨ Features

- 🐍 **Zero External Dependencies**: Standard Python 3 standard library implementation.
- ⚡ **Seamless Helix Pipe Integration**: Uses Helix's `:pipe` (`|`) to send visual selections directly to Ollama and replace them in-place with clean AI-generated code.
- 🧼 **Smart Code Fence Stripping**: Automatically removes markdown wrappers (```python ... ```) so output can be pasted directly into Helix buffers without manual cleanup.
- 🎯 **Preset Actions**: Built-in modes for common editing workflows:
  - `edit` / `refactor`: Refactor code based on user prompt.
  - `fix`: Identify and fix syntax errors, bugs, or logic issues.
  - `explain`: Generate detailed code explanations (preserves markdown formatting).
  - `docs`: Auto-generate docstrings, comments, and type hints.
  - `complete`: Complete code logic from cursor context.
  - `generate`: Generate new code snippets from scratch (perfect for `:append-output` or `:insert-output`).
- 🤖 **Smart Model Resolution**: Automatically detects installed local models (`qwen2.5-coder`, `deepseek-r1`, `codellama`, etc.) or allows specifying custom models.
- ⚙️ **Configurable**: Config file support (`~/.config/hx-ollama/config.json`) for custom hosts, models, temperatures, and timeouts.

---

## 📦 Safe & Interactive Installation

`hx-ollama` includes a safe, interactive installer (`./install.sh`) that asks for your confirmation before modifying any directory or file, showing exact target paths and content diffs.

```bash
git clone https://github.com/your-username/hx-ollama.git
cd hx-ollama

# Optional: Run a dry run first to inspect what will change without modifying anything:
./install.sh --dry-run

# Run the interactive setup:
./install.sh
```

### What the Installer Does (With Your Permission):
1. **Installs Binary**: Copies `hx-ollama` executable to `~/.local/bin/hx-ollama` (or via `pipx`).
2. **Creates Configuration**: Places default config at `~/.config/hx-ollama/config.json` (will **never** overwrite if it already exists).
3. **Appends Keybindings**: Displays the exact `Space + o` TOML keybinding snippet and asks before appending to `~/.config/helix/config.toml` (will **never** duplicate if already added).

---

## ⌨️ Helix Editor Configuration

Run `hx-ollama setup-helix` to print recommended configuration, or add the following to your Helix config file (`~/.config/helix/config.toml`):

```toml
# ==============================================================================
# Helix Editor + Ollama AI Integration (hx-ollama)
# ==============================================================================

# Normal Mode Shortcuts (Space + o for Ollama)
# Uses Macro syntax (@) to open prompt pre-filled so you can type your instruction!
[keys.normal.space.o]
g = "@:append-output<space>hx-ollama<space>generate<space>"
i = "@:insert-output<space>hx-ollama<space>generate<space>"
m = ":sh hx-ollama models"

# Visual / Selection Mode Shortcuts (Space + o for Ollama)
[keys.select.space.o]
# '@|' simulates pressing '|' and pre-fills 'hx-ollama edit ', leaving cursor open for your prompt
e = "@|hx-ollama edit<space>"
f = ":pipe hx-ollama fix"
x = ":pipe hx-ollama explain"
d = ":pipe hx-ollama docs"
c = ":pipe hx-ollama complete"
```

---

## 🚀 How to Use in Helix

### 1. Refactor / Edit Selection (`:pipe`)
1. In visual mode, select a block of code.
2. Press `|` (or Space + `a` + `e`) and type your prompt:
   ```text
   :pipe hx-ollama edit "convert this function to async and add error handling"
   ```
3. Press `Enter`. The selected code will be replaced with the updated code!

### 2. Fix Code Bugs (`:pipe`)
1. Select buggy code in visual mode.
2. Type:
   ```text
   :pipe hx-ollama fix
   ```
3. The selection will be replaced with corrected, bug-free code.

### 3. Explain Code (`:pipe`)
1. Select a complex function in visual mode.
2. Type:
   ```text
   :pipe hx-ollama explain
   ```
3. An explanation of the selected code will be inserted after the selection.

### 4. Append / Insert New Code (`:append-output` / `:insert-output`)
1. In normal mode, type:
   ```text
   :append-output hx-ollama generate "write a python function to calculate fibonacci sequence"
   ```
2. The generated code will be inserted directly at your cursor position!

---

## ⚙️ Configuration (`~/.config/hx-ollama/config.json`)

You can create or modify `~/.config/hx-ollama/config.json` to customize default behavior:

```json
{
  "host": "http://localhost:11434",
  "model": "qwen2.5-coder",
  "temperature": 0.2,
  "timeout": 60,
  "preferred_models": [
    "qwen2.5-coder",
    "deepseek-r1",
    "codellama",
    "llama3.2"
  ]
}
```

---

## 🛠️ CLI Reference & Commands

```bash
# Check installed Ollama models
hx-ollama models

# Print Helix keybindings configuration
hx-ollama setup-helix

# Generate code from terminal prompt
hx-ollama generate "write a rust json parser function"

# Refactor code from stdin
cat main.py | hx-ollama edit "add docstrings and type annotations"
```

---

## 💡 Troubleshooting

- **Error: Ollama is not running**: Start Ollama by running `ollama serve` or opening the Ollama desktop app.
- **Error: No models found**: Download a local code model using `ollama pull qwen2.5-coder`.
