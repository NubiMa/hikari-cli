#!/usr/bin/env bash
set -euo pipefail

LOG_FILE="${1:-system_check.log}"

exec > >(tee "${LOG_FILE}") 2>&1

echo "============================================================"
echo "               HIKARI SYSTEM CHECK REPORT                   "
echo "============================================================"
echo "Timestamp: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
echo "Host: $(uname -a)"
echo ""

echo "--- 1. Prerequisites Check ---"
if command -v go >/dev/null 2>&1; then
    echo "[PASS] Go compiler found: $(go version)"
else
    echo "[FAIL] Go compiler not found in PATH"
    exit 1
fi

if command -v make >/dev/null 2>&1; then
    echo "[PASS] Make utility found: $(make --version | head -n 1)"
else
    echo "[WARN] Make not found; manual go build can be used"
fi

if command -v curl >/dev/null 2>&1; then
    echo "[PASS] curl found: $(curl --version | head -n 1)"
else
    echo "[WARN] curl not found"
fi

echo ""
echo "--- 2. Go Module & Test Suite ---"
echo "Running 'go test -v ./...' (note: ./... is required to test all subpackages):"
go test -v ./...

echo ""
echo "--- 3. Build & Binary Compilation ---"
echo "Compiling Hikari via 'make build'..."
make build

if [ -f "./hikari" ]; then
    echo "[PASS] Binary './hikari' built successfully."
    ls -lh ./hikari
else
    echo "[FAIL] Binary './hikari' not found after build."
    exit 1
fi

echo ""
echo "--- 4. CLI Flags & Version Check ---"
echo "Testing './hikari -v':"
./hikari -v

echo ""
echo "Testing './hikari --help':"
./hikari --help

echo ""
echo "--- 5. Configuration Bootstrap Check ---"
TEMP_CONF_DIR=$(mktemp -d)
trap 'rm -rf "${TEMP_CONF_DIR}"' EXIT

echo "Testing config initialization in temporary dir: ${TEMP_CONF_DIR}"
HIKARI_CONFIG_DIR="${TEMP_CONF_DIR}" ./hikari --init

if [ -f "${TEMP_CONF_DIR}/config.toml" ]; then
    echo "[PASS] Default config.toml created successfully."
    echo "Content sample:"
    head -n 20 "${TEMP_CONF_DIR}/config.toml"
else
    echo "[FAIL] config.toml was not generated."
    exit 1
fi

echo ""
echo "--- 6. Local Provider Health Check (Ollama) ---"
if curl -s "http://127.0.0.1:11434/api/tags" >/dev/null 2>&1; then
    echo "[INFO] Local Ollama service is RUNNING at http://127.0.0.1:11434"
else
    echo "[INFO] Local Ollama service is not running on port 11434 (normal if not started yet)."
fi

echo ""
echo "============================================================"
echo "[SUCCESS] Hikari system check passed with all components ok."
echo "============================================================"
