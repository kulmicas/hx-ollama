import json
import urllib.request
import urllib.error
import sys
from typing import List, Dict, Any, Optional

class OllamaClient:
    def __init__(self, host: str = "http://localhost:11434", timeout: int = 60):
        self.host = host.rstrip("/")
        self.timeout = timeout

    def check_connection(self) -> bool:
        """Returns True if Ollama service is reachable."""
        try:
            req = urllib.request.Request(f"{self.host}/api/tags", method="GET")
            with urllib.request.urlopen(req, timeout=5) as resp:
                return resp.status == 200
        except Exception:
            return False

    def list_models(self) -> List[str]:
        """Returns list of installed model names in Ollama."""
        try:
            url = f"{self.host}/api/tags"
            req = urllib.request.Request(url, method="GET")
            with urllib.request.urlopen(req, timeout=5) as resp:
                if resp.status == 200:
                    data = json.loads(resp.read().decode("utf-8"))
                    return [m["name"] for m in data.get("models", [])]
        except Exception as e:
            print(f"[hx-ollama] Warning: Failed to list models from {self.host}: {e}", file=sys.stderr)
        return []

    def resolve_model(self, requested_model: Optional[str], preferred_models: List[str]) -> str:
        """
        Resolves model to use.
        If requested_model is specified, uses it.
        Otherwise checks installed models and picks the best matching preferred model or first available.
        """
        installed = self.list_models()
        if not installed:
            if requested_model:
                return requested_model
            raise RuntimeError(
                f"No models found on Ollama server at {self.host}.\n"
                f"Please pull a model using 'ollama pull qwen2.5-coder' or specify a model."
            )

        if requested_model:
            # Check if requested model matches any installed model (or prefix like qwen2.5-coder:latest)
            for m in installed:
                if m == requested_model or m.startswith(f"{requested_model}:"):
                    return m
            return requested_model

        # Search for preferred coding models in installed models
        for pref in preferred_models:
            for inst in installed:
                if inst == pref or inst.startswith(f"{pref}:") or pref in inst:
                    return inst

        # Default to first installed model
        return installed[0]

    def generate(
        self,
        model: str,
        prompt: str,
        system: Optional[str] = None,
        temperature: float = 0.2,
        stream: bool = False,
    ) -> str:
        """Sends generate request to Ollama /api/generate endpoint."""
        url = f"{self.host}/api/generate"
        payload = {
            "model": model,
            "prompt": prompt,
            "stream": False,  # Non-streaming for clean editor buffer replacement
            "options": {
                "temperature": temperature,
            }
        }
        if system:
            payload["system"] = system

        data_bytes = json.dumps(payload).encode("utf-8")
        req = urllib.request.Request(
            url,
            data=data_bytes,
            headers={"Content-Type": "application/json"},
            method="POST"
        )

        try:
            with urllib.request.urlopen(req, timeout=self.timeout) as resp:
                if resp.status == 200:
                    response_data = json.loads(resp.read().decode("utf-8"))
                    return response_data.get("response", "")
                else:
                    raise RuntimeError(f"Ollama returned HTTP {resp.status}")
        except urllib.error.URLError as e:
            raise RuntimeError(
                f"Could not connect to Ollama at {self.host}.\n"
                f"Ensure Ollama is running (`ollama serve`). Details: {e}"
            )
