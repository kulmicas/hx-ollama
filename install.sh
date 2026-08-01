#!/usr/bin/env bash
set -e

DRY_RUN=false
if [ "$1" = "--dry-run" ] || [ "$1" = "-n" ]; then
    DRY_RUN=true
    echo "🔍 DRY RUN MODE: No files will be modified."
    echo ""
fi

echo "================================================================="
echo "   hx-ollama Interactive Setup: Helix + Ollama Integration"
echo "================================================================="
echo "This installer will help you safely set up hx-ollama."
echo "You will be prompted before any file or directory is modified."
echo ""

prompt_confirm() {
    local prompt_msg="$1"
    if [ "$DRY_RUN" = true ]; then
        echo "   [Dry Run] Would ask: ${prompt_msg} [y/N]"
        return 1
    fi
    read -p "❓ ${prompt_msg} [y/N] " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        return 0
    else
        return 1
    fi
}

# --- STEP 1: Binary Installation ---
INSTALL_DIR="${HOME}/.local/bin"
TARGET_BIN="${INSTALL_DIR}/hx-ollama"

echo "-----------------------------------------------------------------"
echo "STEP 1: Binary / Executable Installation"
echo "Target path: ${TARGET_BIN}"
echo "-----------------------------------------------------------------"

if prompt_confirm "Do you want to install hx-ollama to ${INSTALL_DIR}?"; then
    if [ "$DRY_RUN" = false ]; then
        mkdir -p "${INSTALL_DIR}"
        echo "⚡ Copying standalone executable to ${TARGET_BIN}..."
        cp bin/hx-ollama "${TARGET_BIN}"
        chmod +x "${TARGET_BIN}"
        echo "✅ Installed binary successfully to ${TARGET_BIN}."
    fi
else
    echo "⏭️  Skipped binary installation."
fi
echo ""

# --- STEP 2: hx-ollama Config File ---
CONFIG_DIR="${HOME}/.config/hx-ollama"
CONFIG_FILE="${CONFIG_DIR}/config.json"

echo "-----------------------------------------------------------------"
echo "STEP 2: Configuration File Setup"
echo "Target path: ${CONFIG_FILE}"
echo "-----------------------------------------------------------------"

if [ -f "${CONFIG_FILE}" ]; then
    echo "ℹ️  Config file already exists at ${CONFIG_FILE}. It will NOT be overwritten."
else
    if prompt_confirm "Do you want to create a default configuration file at ${CONFIG_FILE}?"; then
        if [ "$DRY_RUN" = false ]; then
            mkdir -p "${CONFIG_DIR}"
            cat << 'EOF' > "${CONFIG_FILE}"
{
  "host": "http://localhost:11434",
  "model": "",
  "temperature": 0.2,
  "timeout": 60
}
EOF
            echo "✅ Created default config at ${CONFIG_FILE}."
        fi
    else
        echo "⏭️  Skipped creating default config file."
    fi
fi
echo ""

# --- STEP 3: Helix Config Integration ---
HELIX_CONFIG_DIR="${HOME}/.config/helix"
HELIX_CONFIG_FILE="${HELIX_CONFIG_DIR}/config.toml"

echo "-----------------------------------------------------------------"
echo "STEP 3: Helix Keybindings Setup"
echo "Target path: ${HELIX_CONFIG_FILE}"
echo "Snippet to append:"
echo "-----------------------------------------------------------------"
cat << 'EOF'
# ==============================================================================
# Helix Editor + Ollama AI Integration (hx-ollama)
# ==============================================================================

[keys.normal.space.o]
g = "@:append-output<space>hx-ollama<space>generate<space>"
i = "@:insert-output<space>hx-ollama<space>generate<space>"
m = ":sh hx-ollama models"

[keys.select.space.o]
e = "@|hx-ollama edit<space>"
f = ":pipe hx-ollama fix"
x = ":pipe hx-ollama explain"
d = ":pipe hx-ollama docs"
c = ":pipe hx-ollama complete"
EOF
echo "-----------------------------------------------------------------"

if [ -f "${HELIX_CONFIG_FILE}" ] && grep -q "hx-ollama" "${HELIX_CONFIG_FILE}"; then
    echo "ℹ️  hx-ollama keybindings are already present in ${HELIX_CONFIG_FILE}."
else
    if prompt_confirm "Do you want to append these Space + o keybindings to ${HELIX_CONFIG_FILE}?"; then
        if [ "$DRY_RUN" = false ]; then
            mkdir -p "${HELIX_CONFIG_DIR}"
            python3 bin/hx-ollama install-helix
        fi
    else
        echo "⏭️  Skipped updating Helix config."
    fi
fi

echo ""
echo "================================================================="
echo "🎉 Setup complete! All steps finished."
echo "================================================================="
