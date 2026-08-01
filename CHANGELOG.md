# Changelog

All notable changes to the `hx-ollama` project will be documented in this file.

## [1.0.0] - 2026-08-02

### Added
- **Pure Go Single Static Binary**: Zero-dependency Go implementation with `net/http` and `encoding/json`.
- **Helix Integration**: Native support for Helix `:pipe` (`|`), `:append-output`, and `:insert-output`.
- **Space + o Keybindings**: Configured `Space + o` keybinding namespace for normal and visual visual selections (`edit`, `fix`, `explain`, `docs`, `complete`, `generate`, `models`).
- **Fail-Safe Protection**: Preserves original code selection when Ollama is offline or returns API errors.
- **LAN & Remote Ollama Support**: Connects seamlessly to local or LAN network Ollama hosts via `OLLAMA_HOST` or `~/.config/hx-ollama/config.json`.
- **Automatic Template Config Generation**: Automatically creates `~/.config/hx-ollama/config.json` with explanatory comments.
- **Smart Code Fence Stripping**: Strips markdown code fence wrappers (` ```python ... ``` `) for clean buffer replacement.
