import re

def strip_code_fences(text: str) -> str:
    """
    Strips top-level markdown code fence wrappers (e.g., ```python ... ```)
    so Helix receives raw, clean code ready for buffer insertion/replacement.
    """
    if not text:
        return text

    lines = text.strip().splitlines()

    # Check if text is enclosed in ```...```
    if len(lines) >= 2 and lines[0].startswith("```") and lines[-1].strip() == "```":
        lines = lines[1:-1]
        return "\n".join(lines) + ("\n" if text.endswith("\n") else "")

    # Fallback regex for code block anywhere in text if response contained chatter + code block
    code_block_match = re.search(r"```(?:\w+)?\n(.*?)```", text, re.DOTALL)
    if code_block_match:
        return code_block_match.group(1)

    return text

def format_output(response_text: str, code_only: bool = True) -> str:
    """
    Formats LLM response text based on mode.
    If code_only is True, strips markdown fences and conversational intro/outro.
    """
    if code_only:
        return strip_code_fences(response_text)
    return response_text
