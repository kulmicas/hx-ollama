import sys
import os
import argparse
from typing import Optional

from .config import load_config, save_config, CONFIG_FILE
from .ollama import OllamaClient
from .formatter import format_output
from .prompts import (
    SYSTEM_PROMPT_EDIT,
    SYSTEM_PROMPT_FIX,
    SYSTEM_PROMPT_EXPLAIN,
    SYSTEM_PROMPT_DOCS,
    SYSTEM_PROMPT_COMPLETE,
    SYSTEM_PROMPT_GENERATE,
)

HELIX_CONFIG_SNIPPET = """
# ==============================================================================
# Helix Editor + Ollama AI Integration (hx-ollama)
# ==============================================================================

# Normal Mode Keybindings (Space + o for Ollama)
[keys.normal.space.o]
# Uses Macro syntax (@) to open prompt pre-filled so you can type your instruction!
g = "@:append-output<space>hx-ollama<space>generate<space>"
i = "@:insert-output<space>hx-ollama<space>generate<space>"
m = ":sh hx-ollama models"

# Visual / Selection Mode Keybindings (Space + o for Ollama)
[keys.select.space.o]
# Opens pipe prompt pre-filled with 'hx-ollama edit ', leaving cursor open for your instruction!
e = "@|hx-ollama edit<space>"
f = ":pipe hx-ollama fix"
x = ":pipe hx-ollama explain"
d = ":pipe hx-ollama docs"
c = ":pipe hx-ollama complete"
"""

import select

def read_stdin(timeout: float = 0.05) -> str:
    """Reads stdin if data is ready to be read immediately."""
    if not sys.stdin.isatty():
        try:
            rlist, _, _ = select.select([sys.stdin], [], [], timeout)
            if rlist:
                return sys.stdin.read()
        except Exception:
            return ""
    return ""

def main():
    config = load_config()

    parser = argparse.ArgumentParser(
        prog="hx-ollama",
        description="Helix Editor + Ollama AI Integration Tool via Pipe (|) and Append/Insert Output",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  Visual Pipe in Helix:
    :pipe hx-ollama edit "convert function to async"
    :pipe hx-ollama fix
    :pipe hx-ollama explain
    :pipe hx-ollama docs

  Append / Insert in Helix:
    :append-output hx-ollama generate "write a python quicksort function"

  CLI / Terminal:
    hx-ollama models
    hx-ollama setup-helix
        """
    )

    parser.add_argument(
        "action_or_prompt",
        nargs="*",
        help="Action (edit, fix, explain, docs, complete, generate, models, setup-helix) or prompt instruction"
    )
    parser.add_argument("-m", "--model", help="Specify Ollama model to use")
    parser.add_argument("-t", "--temperature", type=float, help="Temperature setting (0.0 to 1.0)")
    parser.add_argument("--host", help="Ollama host URL (default: http://localhost:11434)")
    parser.add_argument("--raw", action="store_true", help="Force raw code output (strip code fences)")
    parser.add_argument("--markdown", action="store_true", help="Preserve markdown output (do not strip code fences)")
    parser.add_argument("--keep-code", action="store_true", help="Preserve original piped code and append response below it")

    args = parser.parse_args()

    # Read stdin if piped from Helix
    stdin_content = read_stdin()

    # Parse positional action / prompt
    positional = args.action_or_prompt
    action = positional[0].lower() if positional else ""
    extra_prompt = " ".join(positional[1:]) if len(positional) > 1 else ""

    # Check for direct helper commands
    if action == "models":
        client = OllamaClient(host=args.host or config["host"])
        if not client.check_connection():
            print(f"[hx-ollama] Error: Ollama is not running at {client.host}. Please start Ollama (`ollama serve`).", file=sys.stderr)
            sys.exit(1)
        models = client.list_models()
        if models:
            print("Installed Ollama Models:")
            for m in models:
                print(f"  - {m}")
        else:
            print("No Ollama models found. Pull a model using: `ollama pull qwen2.5-coder`")
        return

    if action == "setup-helix":
        print("Copy and paste the following snippet into your Helix config (~/.config/helix/config.toml):")
        print(HELIX_CONFIG_SNIPPET)
        return

    if action in ("install-helix", "init-helix"):
        helix_config_dir = os.path.expanduser("~/.config/helix")
        helix_config_file = os.path.join(helix_config_dir, "config.toml")
        os.makedirs(helix_config_dir, exist_ok=True)

        existing_content = ""
        if os.path.exists(helix_config_file):
            with open(helix_config_file, "r", encoding="utf-8") as f:
                existing_content = f.read()

        if "hx-ollama" in existing_content:
            print(f"[hx-ollama] Notice: hx-ollama configuration is already present in {helix_config_file}")
            return

        with open(helix_config_file, "a", encoding="utf-8") as f:
            if existing_content and not existing_content.endswith("\n"):
                f.write("\n")
            f.write(HELIX_CONFIG_SNIPPET)

        print(f"✅ Successfully appended hx-ollama keybindings to {helix_config_file}!")
        return

    # Determine command mode & system prompt
    code_only = True
    keep_code = args.keep_code
    system_prompt = SYSTEM_PROMPT_EDIT
    user_prompt = ""

    if action == "fix":
        system_prompt = SYSTEM_PROMPT_FIX
        user_prompt = extra_prompt or "Fix any bugs, syntax errors, or logical issues in this code."
    elif action == "explain":
        system_prompt = SYSTEM_PROMPT_EXPLAIN
        user_prompt = extra_prompt or "Explain this code in detail."
        code_only = False
        keep_code = True  # Preserve original code selection so explain doesn't wipe out code in Helix
    elif action == "docs":
        system_prompt = SYSTEM_PROMPT_DOCS
        user_prompt = extra_prompt or "Add clear docstrings and comments to this code."
    elif action == "complete":
        system_prompt = SYSTEM_PROMPT_COMPLETE
        user_prompt = extra_prompt or "Complete the logic for this code snippet."
    elif action == "generate" or action == "create":
        system_prompt = SYSTEM_PROMPT_GENERATE
        user_prompt = extra_prompt or " ".join(positional)
    elif action in ("edit", "refactor", "change", "transform"):
        system_prompt = SYSTEM_PROMPT_EDIT
        user_prompt = extra_prompt or "Refactor and clean up this code."
    else:
        # Default fallback: if action was passed, treat full positional arguments as user prompt
        full_instruction = " ".join(positional)
        if stdin_content:
            system_prompt = SYSTEM_PROMPT_EDIT
            user_prompt = full_instruction or "Improve and refine this code."
        else:
            system_prompt = SYSTEM_PROMPT_GENERATE
            user_prompt = full_instruction or "Generate the requested code."

    # Combine stdin with user prompt if stdin exists
    if stdin_content:
        full_prompt = f"User Request: {user_prompt}\n\nCode Context:\n{stdin_content}"
    else:
        full_prompt = user_prompt

    if not full_prompt.strip():
        parser.print_help(sys.stderr)
        sys.exit(1)

    # CLI flag overrides
    if args.raw:
        code_only = True
    elif args.markdown:
        code_only = False

    host = args.host or config.get("host", "http://localhost:11434")
    temp = args.temperature if args.temperature is not None else config.get("temperature", 0.2)
    timeout = config.get("timeout", 60)

    client = OllamaClient(host=host, timeout=timeout)

    # Resolve model
    req_model = args.model or config.get("model") or None
    try:
        model = client.resolve_model(req_model, config.get("preferred_models", []))
    except Exception as e:
        print(f"[hx-ollama] Error: {e}", file=sys.stderr)
        sys.exit(1)

    try:
        raw_response = client.generate(
            model=model,
            prompt=full_prompt,
            system=system_prompt,
            temperature=temp,
        )
        formatted = format_output(raw_response, code_only=code_only)

        # If keep_code is enabled and stdin_content exists, preserve original code above the explanation
        if keep_code and stdin_content:
            formatted = f"{stdin_content.rstrip()}\n\n---\n### 💡 Code Explanation\n{formatted}\n"

        print(formatted, end="")
    except Exception as e:
        print(f"[hx-ollama] Error executing Ollama request: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
