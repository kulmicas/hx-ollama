import unittest
from unittest.mock import MagicMock, patch
import json
import io
import sys

from hx_ollama.formatter import strip_code_fences, format_output
from hx_ollama.config import DEFAULT_CONFIG, load_config
from hx_ollama.ollama import OllamaClient
from hx_ollama.prompts import (
    SYSTEM_PROMPT_EDIT,
    SYSTEM_PROMPT_FIX,
    SYSTEM_PROMPT_EXPLAIN,
    SYSTEM_PROMPT_DOCS,
    SYSTEM_PROMPT_GENERATE,
    SYSTEM_PROMPT_COMPLETE,
)
from hx_ollama.cli import read_stdin

class TestFormatter(unittest.TestCase):
    def test_strip_code_fences_simple(self):
        input_text = "```python\ndef foo():\n    return 42\n```"
        expected = "def foo():\n    return 42"
        self.assertEqual(strip_code_fences(input_text).strip(), expected)

    def test_strip_code_fences_no_lang(self):
        input_text = "```\nconst x = 10;\n```"
        expected = "const x = 10;"
        self.assertEqual(strip_code_fences(input_text).strip(), expected)

    def test_strip_code_fences_with_chatter(self):
        input_text = "Here is the refactored code:\n\n```python\ndef bar():\n    pass\n```\nHope this helps!"
        expected = "def bar():\n    pass\n"
        self.assertEqual(strip_code_fences(input_text), expected)

    def test_strip_code_fences_raw_text(self):
        input_text = "def bar():\n    pass"
        self.assertEqual(strip_code_fences(input_text), input_text)

    def test_format_output_code_only(self):
        input_text = "```rust\nfn main() {}\n```"
        self.assertEqual(format_output(input_text, code_only=True).strip(), "fn main() {}")

    def test_format_output_markdown_preserved(self):
        input_text = "Here is the explanation:\n```rust\nfn main() {}\n```"
        self.assertEqual(format_output(input_text, code_only=False), input_text)

class TestOllamaClient(unittest.TestCase):
    @patch("hx_ollama.ollama.OllamaClient.list_models")
    def test_resolve_model_explicit(self, mock_list):
        mock_list.return_value = ["qwen2.5-coder:latest", "llama3.2:latest"]
        client = OllamaClient()
        self.assertEqual(client.resolve_model("qwen2.5-coder", []), "qwen2.5-coder:latest")

    @patch("hx_ollama.ollama.OllamaClient.list_models")
    def test_resolve_model_fallback_preferred(self, mock_list):
        mock_list.return_value = ["codellama:7b", "llama3:latest"]
        client = OllamaClient()
        preferred = ["qwen2.5-coder", "codellama", "llama3"]
        self.assertEqual(client.resolve_model(None, preferred), "codellama:7b")

    @patch("hx_ollama.ollama.OllamaClient.list_models")
    def test_resolve_model_fallback_first(self, mock_list):
        mock_list.return_value = ["custom-model:latest"]
        client = OllamaClient()
        self.assertEqual(client.resolve_model(None, ["qwen2.5-coder"]), "custom-model:latest")

    @patch("hx_ollama.ollama.OllamaClient.list_models")
    def test_resolve_model_empty_raises(self, mock_list):
        mock_list.return_value = []
        client = OllamaClient()
        with self.assertRaises(RuntimeError):
            client.resolve_model(None, [])

    @patch("urllib.request.urlopen")
    def test_generate_success(self, mock_urlopen):
        mock_resp = MagicMock()
        mock_resp.status = 200
        mock_resp.read.return_value = json.dumps({"response": "def baz(): pass"}).encode("utf-8")
        mock_urlopen.return_value.__enter__.return_value = mock_resp

        client = OllamaClient()
        output = client.generate("test-model", "test prompt", system="test system")
        self.assertEqual(output, "def baz(): pass")

class TestPrompts(unittest.TestCase):
    def test_system_prompts_exist(self):
        self.assertIn("Helix", SYSTEM_PROMPT_EDIT)
        self.assertIn("CRITICAL RULE", SYSTEM_PROMPT_EDIT)
        self.assertIn("CRITICAL RULE", SYSTEM_PROMPT_FIX)
        self.assertIn("CRITICAL RULE", SYSTEM_PROMPT_DOCS)
        self.assertIn("CRITICAL RULE", SYSTEM_PROMPT_COMPLETE)
        self.assertIn("CRITICAL RULE", SYSTEM_PROMPT_GENERATE)

class TestConfig(unittest.TestCase):
    def test_default_config_fields(self):
        self.assertIn("host", DEFAULT_CONFIG)
        self.assertIn("temperature", DEFAULT_CONFIG)
        self.assertIn("preferred_models", DEFAULT_CONFIG)

if __name__ == "__main__":
    unittest.main()
