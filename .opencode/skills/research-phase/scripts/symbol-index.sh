#!/bin/bash
# symbol-index.sh - Extract and index public symbols from codebase
# High-compatibility version using grep/ast-grep

set -euo pipefail
SEARCH_PATH="${1:-.}"

extract_symbols() {
    local path="$1"

    echo "=== TypeScript/JS Exports ===" >&2
    grep -rnE "export (function|const|class|interface|type) [A-Z]" "$path" 2>/dev/null | head -100 || true

    echo ""
    echo "=== Python Public Symbols ===" >&2
    grep -rnE "^(def|class) [a-zA-Z0-9]" "$path" 2>/dev/null | grep -vE "^(def|class) _" | head -100 || true

    echo ""
    echo "=== Golang Public Symbols ===" >&2
    # Match public functions (capital letter)
    grep -rnE "^func [A-Z]" "$path" 2>/dev/null | head -100 || true
    # Match public types/structs/interfaces
    grep -rnE "^type [A-Z]" "$path" 2>/dev/null | head -100 || true
}

extract_symbols "$SEARCH_PATH"
