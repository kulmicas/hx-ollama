# `hx-ollama`: Pure C Local/LAN AI Integration for Helix Editor

`hx-ollama` is a **35 KB, ultra-fast (<0.2ms startup), zero-dependency static binary** written in pure C. It connects local or LAN-hosted **Ollama AI models** with **Helix Editor (`hx`)** via Helix's native `:pipe` (`|`), `:append-output`, and `:insert-output` features.

---

## ✨ Key Highlights

- ⚡ **Ultra-Fast & Lightweight**: ~35 KB static binary, starts in `< 0.2ms`.
- 🐍 **Zero External Dependencies**: Built with POSIX Sockets (`sys/socket.h`) and STB-style `cJSON`. No Python, Node, Go, or external HTTP libraries required.
- 🌐 **Local & LAN AI Support**: Easily connects to Ollama running locally or on another machine on your network (`http://192.168.x.x:11434` or `OLLAMA_HOST`).
- 🧼 **Smart Code Fence Stripping**: Automatically removes markdown code fence wrappers (```python ... ```) for clean in-place buffer replacement.
- 💡 **Preserves Code on Explanation**: `explain` appends structured explanations below your code without wiping it out.

---

## ⚡ Building & Installing

### 1. Build & Install (1 Command)
```bash
make install
```
*(This compiles `hx-ollama.c` and copies the 35 KB binary to `~/.local/bin/hx-ollama`).*

Or compile directly with `gcc` or `clang`:
```bash
gcc -O3 hx-ollama.c cJSON.c -o ~/.local/bin/hx-ollama
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

You can configure your local or LAN Ollama endpoint in `~/.config/hx-ollama/config.json`:

```json
{
  "host": "http://localhost:11434",
  "model": "qwen2.5-coder:14b-instruct"
}
```

*Or temporarily set a remote LAN AI host via environment variable:*
```bash
export OLLAMA_HOST="http://192.168.1.100:11434"
```

---

## 🚀 How to Use in Helix

- **Refactor Selection**: Select text in visual mode (`v`) $\rightarrow$ `Space + o + e` $\rightarrow$ type `convert to async` $\rightarrow$ `Enter`.
- **Auto-Fix Bugs**: Select code in visual mode (`v`) $\rightarrow$ `Space + o + f`.
- **Explain Code**: Select code in visual mode (`v`) $\rightarrow$ `Space + o + x`. Appends explanation below code.
- **Generate New Code**: In normal mode $\rightarrow$ `Space + o + g` $\rightarrow$ type `write a python json parser` $\rightarrow$ `Enter`.
- **List Installed AI Models**: Type `:sh hx-ollama models`.

---

## 📄 License
MIT License.
