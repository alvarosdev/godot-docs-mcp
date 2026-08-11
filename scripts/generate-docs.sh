#!/usr/bin/env bash
set -euo pipefail

# generate-docs.sh — Download Godot docs, convert RST → GFM, produce versioned artifact.
# Converts RST to GitHub-Flavored Markdown via pandoc for LLM-friendly output.
# Failed conversions preserve the original .rst as a fallback.
# Runs in CI (ubuntu-latest with pandoc, curl, jq, tree, unzip installed).
#
# Usage: scripts/generate-docs.sh "3.6,4.7"
#        scripts/generate-docs.sh "4.7"

DOCS_DIR="docs"
GODOT_DOCS_REPO="https://github.com/godotengine/godot-docs"
MAX_WORKERS=8
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

main() {
    local versions_str="$1"
    local -a processed=()

    for version in $(echo "$versions_str" | tr ',' ' '); do
        version=$(echo "$version" | xargs)
        echo ""
        echo "============================================"
        echo "Processing Godot $version documentation"
        echo "============================================"
        process_version "$version"
        processed+=("$version")
    done

    echo ""
    echo "============================================"
    echo "Generating versions.json"
    echo "============================================"
    generate_versions_json "${processed[@]}"

    echo ""
    echo "Done. Processed ${#processed[@]} versions."
}

process_version() {
    local version="$1"
    local version_dir="$DOCS_DIR/$version"

    # 1. Download ZIP from GitHub.
    echo "  Downloading godot-docs branch: $version..."
    local zip_file="$TEMP_DIR/godot-docs-$version.zip"
    curl -fsSL "$GODOT_DOCS_REPO/archive/refs/heads/$version.zip" -o "$zip_file"

    # 2. Extract.
    echo "  Extracting..."
    local extract_dir="$TEMP_DIR/extract-$version"
    mkdir -p "$extract_dir"
    unzip -q "$zip_file" -d "$extract_dir"

    # The ZIP contains a top-level directory like "godot-docs-4.7".
    # Move its contents into docs/$version.
    mkdir -p "$version_dir"
    local zip_root
    zip_root=$(echo "$extract_dir"/godot-docs-*)
    if [[ -d "$zip_root" ]]; then
        mv "$zip_root"/* "$version_dir"/ 2>/dev/null || true
        mv "$zip_root"/.[!.]* "$version_dir"/ 2>/dev/null || true
    else
        echo "  WARNING: could not find extracted docs directory in $extract_dir"
        return 1
    fi

    # 3. Cleanup: remove Sphinx/Python/build artifacts.
    #    RST files are served directly — no pandoc conversion needed.
    echo "  Cleaning up..."
    local trash_dirs=(
        "_extensions" "_static" "_templates" "_tools"
        ".github"
    )
    local trash_files=(
        "conf.py" "Makefile" "make.bat" "pyproject.toml"
        "requirements.txt" ".readthedocs.yml"
        ".gitattributes" ".git-blame-ignore-revs" ".gitignore"
        ".editorconfig" ".mailmap" ".lycheeignore"
        ".pre-commit-config.yaml"
        "404.rst" "AUTHORS.md" "robots.txt" "README.md"
    )
    for d in "${trash_dirs[@]}"; do
        rm -rf "$version_dir/$d" 2>/dev/null || true
    done
    for f in "${trash_files[@]}"; do
        rm -f "$version_dir/$f" 2>/dev/null || true
    done

    # 4. Convert RST → GFM Markdown (parallel, capped at MAX_WORKERS).
    #    - Strip Sphinx ".. table::" / ":widths:" directives so grid tables render.
    #    - Post-clean: drop "classref-*" lines and "<div class=\"rst-class\">" blocks.
    #    - Delete the source .rst only on successful conversion; keep it as a
    #      fallback when pandoc fails.
    local rst_count
    rst_count=$(find "$version_dir" -name '*.rst' | wc -l)
    echo "  Converting $rst_count RST files to GFM (max $MAX_WORKERS workers)..."
    find "$version_dir" -name '*.rst' -print0 \
        | xargs -0 -P "$MAX_WORKERS" -I {} bash -c '
            sed "/^\.\. table::$/d; /^   :widths:/d" "$1" \
                | pandoc -f rst -t gfm --wrap=none 2>/dev/null \
                | sed "/^<div class=\"rst-class\">$/,/^<\/div>$/d; /^classref-/d; /:::: {#/,/^::::$/d" \
                > "${1%.rst}.md" \
            && rm "$1" \
            || { rm -f "${1%.rst}.md"; echo "  [SKIP] $1"; }
        ' _ {}

    # 5. Cleanup: preserve .md, .rst (fallback), docs_tree.txt; delete the rest.
    find "$version_dir" -type f ! -name '*.md' ! -name '*.rst' ! -name 'docs_tree.txt' -delete 2>/dev/null || true
    find "$version_dir" -depth -type d -empty -delete 2>/dev/null || true

    # 6. Generate docs tree.
    echo "  Generating docs tree..."
    if command -v tree &>/dev/null; then
        tree "$version_dir" > "$version_dir/docs_tree.txt"
    else
        echo "  WARNING: 'tree' command not found, skipping tree generation"
    fi

    local md_count rst_left
    md_count=$(find "$version_dir" -name '*.md' | wc -l)
    rst_left=$(find "$version_dir" -name '*.rst' | wc -l)
    echo "  Version $version done: $md_count .md files, $rst_left .rst fallback."
}

generate_versions_json() {
    local versions=("$@")

    # Sort versions (ascending by major.minor).
    local sorted
    sorted=$(printf '%s\n' "${versions[@]}" | sort -t. -k1,1n -k2,2n)
    local -a sorted_arr=()
    while IFS= read -r v; do
        [[ -n "$v" ]] && sorted_arr+=("$v")
    done <<< "$sorted"

    # Pick latest (last after sort).
    local latest="${sorted_arr[-1]}"

    # Build date in UTC.
    local built
    built=$(date -u +"%Y.%m.%d")

    # Build JSON.
    local json_versions
    json_versions=$(printf '%s\n' "${sorted_arr[@]}" | jq -R . | jq -s .)
    jq -n \
        --argjson versions "$json_versions" \
        --arg latest "$latest" \
        --arg built "$built" \
        '{versions: $versions, latest: $latest, built: $built}' \
        > "$DOCS_DIR/versions.json"

    echo "versions.json written for versions: ${sorted_arr[*]}"
    echo "  latest: $latest"
    echo "  built:  $built"
}

if [[ $# -lt 1 ]]; then
    echo "Usage: $0 <versions>"
    echo "Example: $0 '3.6,4.7'"
    exit 1
fi

main "$1"
