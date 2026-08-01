#!/usr/bin/env bash
set -e

echo "🚀 Installing hx-ollama (Helix + Ollama AI)..."

INSTALL_DIR="${HOME}/.local/bin"
mkdir -p "${INSTALL_DIR}"

if command -v pipx >/dev/null 2>&1; then
    echo "📦 Installing via pipx..."
    pipx install --force .
elif command -v pip3 >/dev/null 2>&1; then
    echo "📦 Installing via pip3..."
    pip3 install --user --break-system-packages .
else
    echo "⚡ Installing standalone script to ${INSTALL_DIR}/hx-ollama..."
    cp bin/hx-ollama "${INSTALL_DIR}/hx-ollama"
    chmod +x "${INSTALL_DIR}/hx-ollama"
fi

echo "⚙️ Setting up Helix config..."
if command -v hx-ollama >/dev/null 2>&1; then
    hx-ollama install-helix
else
    "${INSTALL_DIR}/hx-ollama" install-helix
fi

echo ""
echo "🎉 Setup complete! You can now use Space + o in Helix Editor to run local AI models via Ollama."
