import os
import json
from pathlib import Path
from typing import Dict, Any

CONFIG_DIR = Path.home() / ".config" / "hx-ollama"
CONFIG_FILE = CONFIG_DIR / "config.json"

DEFAULT_CONFIG: Dict[str, Any] = {
    "host": "http://localhost:11434",
    "model": "",  # Empty means auto-detect from local Ollama models
    "temperature": 0.2,
    "timeout": 60,
    "stream": False,
    "preferred_models": [
        "qwen2.5-coder",
        "qwen2.5-coder:7b",
        "qwen2.5-coder:1.5b",
        "deepseek-r1",
        "codellama",
        "llama3.2",
        "llama3.1",
        "mistral",
    ]
}

def load_config() -> Dict[str, Any]:
    """Loads configuration from ~/.config/hx-ollama/config.json if it exists."""
    config = DEFAULT_CONFIG.copy()
    if CONFIG_FILE.exists():
        try:
            with open(CONFIG_FILE, "r", encoding="utf-8") as f:
                user_config = json.load(f)
                config.update(user_config)
        except Exception:
            pass
    return config

def save_config(config: Dict[str, Any]) -> None:
    """Saves configuration to ~/.config/hx-ollama/config.json."""
    CONFIG_DIR.mkdir(parents=True, exist_ok=True)
    with open(CONFIG_FILE, "w", encoding="utf-8") as f:
        json.dump(config, f, indent=2)
