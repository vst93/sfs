#!/usr/bin/env bash

set -euo pipefail

REPO_OWNER="vst93"
REPO_NAME="sfs"
BINARY_NAME="sfs"
REPO_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}"
API_URL="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
FORCE_INSTALL="${FORCE_INSTALL:-0}"
AUTO_DELETE_INSTALL_SCRIPT="${AUTO_DELETE_INSTALL_SCRIPT:-1}"

TEMP_DIR=""
SCRIPT_PATH="${BASH_SOURCE[0]:-}"

log_info() {
    printf '[INFO] %s\n' "$*"
}

log_warn() {
    printf '[WARN] %s\n' "$*" >&2
}

log_error() {
    printf '[ERROR] %s\n' "$*" >&2
}

cleanup() {
    if [ -n "$TEMP_DIR" ]; then
        rm -rf "$TEMP_DIR"
    fi

    if should_self_delete_script; then
        rm -f "$SCRIPT_PATH" || true
    fi
}

trap cleanup EXIT

is_termux() {
    [ -n "${TERMUX_VERSION:-}" ] || [ "${PREFIX:-}" = "/data/data/com.termux/files/usr" ] || [ -d "/data/data/com.termux/files/usr" ]
}

is_interactive() {
    [ -t 0 ]
}

has_cmd() {
    command -v "$1" >/dev/null 2>&1
}

should_self_delete_script() {
    local script_name

    if [ "$AUTO_DELETE_INSTALL_SCRIPT" != "1" ]; then
        return 1
    fi

    if [ -z "$SCRIPT_PATH" ] || [ ! -f "$SCRIPT_PATH" ]; then
        return 1
    fi

    script_name="$(basename "$SCRIPT_PATH")"
    if [ "$script_name" != "install.sh" ]; then
        return 1
    fi

    case "$SCRIPT_PATH" in
        cmd/install.sh|*/cmd/install.sh)
            return 1
            ;;
    esac

    return 0
}

print_help() {
    cat <<EOF
Usage: ${BINARY_NAME}-install [OPTIONS]

Install latest ${BINARY_NAME} release for current platform.

Options:
  -h, --help                Show this help and exit
      --install-dir <dir>   Override install directory
      --force               Continue install when checksum fetch/verify fails

Environment variables:
  INSTALL_DIR                   Same as --install-dir
  FORCE_INSTALL=1               Same as --force
  AUTO_DELETE_INSTALL_SCRIPT=0  Disable auto-deleting downloaded install.sh
EOF
}

parse_args() {
    while [ "$#" -gt 0 ]; do
        case "$1" in
            -h|--help)
                print_help
                exit 0
                ;;
            --install-dir)
                if [ "$#" -lt 2 ]; then
                    log_error "--install-dir requires a directory argument"
                    exit 2
                fi
                INSTALL_DIR="$2"
                shift
                ;;
            --force)
                FORCE_INSTALL="1"
                ;;
            *)
                log_error "Unknown argument: $1"
                log_error "Available options: --help, --install-dir <dir>, --force"
                exit 2
                ;;
        esac
        shift
    done
}

init_temp_dir() {
    TEMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t sfs)"
}

require_download_tool() {
    if has_cmd curl || has_cmd wget; then
        return 0
    fi

    log_error "curl or wget is required for downloading"
    exit 1
}

require_extract_tool() {
    if has_cmd unzip; then
        return 0
    fi

    log_error "unzip is required for extracting release archive"
    if is_termux; then
        log_info "On Termux, run: pkg install unzip"
    fi
    exit 1
}

download_file() {
    local url="$1"
    local output_file="$2"

    if has_cmd curl; then
        curl -fsSL --retry 3 --retry-delay 1 -o "$output_file" "$url"
    elif has_cmd wget; then
        wget -q -O "$output_file" "$url"
    else
        log_error "curl or wget is required for downloading"
        return 1
    fi

    if [ ! -s "$output_file" ]; then
        log_error "Download failed or file is empty: $url"
        return 1
    fi
}

fetch_latest_version() {
    local response
    local version

    if has_cmd curl; then
        response="$(curl -fsSL "$API_URL")"
    elif has_cmd wget; then
        response="$(wget -q -O - "$API_URL")"
    else
        log_error "curl or wget is required to fetch release info"
        exit 1
    fi

    version="$(printf '%s\n' "$response" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"

    if [ -z "$version" ]; then
        log_error "Failed to parse latest version from GitHub API"
        exit 1
    fi

    VERSION="$version"
}

detect_platform() {
    local uname_s uname_m

    uname_s="$(uname -s)"
    uname_m="$(uname -m)"

    case "$uname_s" in
        Darwin)
            OS="darwin"
            ;;
        Linux)
            if is_termux; then
                OS="android"
            else
                OS="linux"
            fi
            ;;
        *)
            log_error "Unsupported OS: $uname_s"
            exit 1
            ;;
    esac

    case "$uname_m" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l|armv8l|arm)
            log_error "32-bit ARM is not supported: $uname_m"
            log_error "Please use arm64 device, or build from source"
            exit 1
            ;;
        *)
            log_error "Unsupported architecture: $uname_m"
            exit 1
            ;;
    esac

    log_info "Detected: ${OS}-${ARCH}"
}

get_download_info() {
    FILENAME="${BINARY_NAME}-${OS}-${ARCH}.zip"
    DOWNLOAD_URL="${REPO_URL}/releases/download/${VERSION}/${FILENAME}"
}

get_sha256_hash() {
    local sha256_url="$1"
    local sha_file="$TEMP_DIR/${FILENAME}.sha256"

    download_file "$sha256_url" "$sha_file"
    awk '{print $1}' "$sha_file"
}

compute_sha256() {
    local file="$1"

    if has_cmd sha256sum; then
        sha256sum "$file" | awk '{print $1}'
    elif has_cmd shasum; then
        shasum -a 256 "$file" | awk '{print $1}'
    elif has_cmd openssl; then
        openssl dgst -sha256 "$file" | awk '{print $NF}'
    else
        return 1
    fi
}

verify_sha256() {
    local file="$1"
    local expected_sha="$2"
    local actual_sha

    if ! actual_sha="$(compute_sha256 "$file")"; then
        log_warn "No SHA256 tool found, skipping verification"
        return 0
    fi

    if [ "$actual_sha" != "$expected_sha" ]; then
        log_error "SHA256 verification failed"
        log_error "Expected: $expected_sha"
        log_error "Actual:   $actual_sha"
        return 1
    fi

    log_info "SHA256 verification passed"
}

prompt_continue_or_abort() {
    local reason="$1"

    if [ "$FORCE_INSTALL" = "1" ]; then
        log_warn "$reason, FORCE_INSTALL=1, continuing"
        return 0
    fi

    if is_interactive; then
        printf '%s, continue? (y/N): ' "$reason"
        read -r reply
        case "$reply" in
            y|Y|yes|YES)
                return 0
                ;;
            *)
                log_error "Installation cancelled"
                exit 1
                ;;
        esac
    else
        log_error "$reason, non-interactive session aborted. Set FORCE_INSTALL=1 to override"
        exit 1
    fi
}

determine_install_dir() {
    if [ -n "${INSTALL_DIR:-}" ]; then
        printf '%s\n' "$INSTALL_DIR"
        return
    fi

    if is_termux && [ -n "${PREFIX:-}" ]; then
        printf '%s\n' "$PREFIX/bin"
        return
    fi

    if [ -d "/usr/local/bin" ] && { [ -w "/usr/local/bin" ] || { ! is_termux && has_cmd sudo; }; }; then
        printf '%s\n' "/usr/local/bin"
        return
    fi

    if [ -n "${HOME:-}" ]; then
        printf '%s\n' "$HOME/.local/bin"
        return
    fi

    log_error "Cannot determine install directory"
    exit 1
}

ensure_dir_exists() {
    local dir="$1"

    if [ -d "$dir" ]; then
        return 0
    fi

    if mkdir -p "$dir" 2>/dev/null; then
        return 0
    fi

    if is_termux; then
        log_error "Cannot create directory: $dir"
        exit 1
    fi

    if has_cmd sudo; then
        sudo mkdir -p "$dir" || {
            log_error "sudo mkdir failed: $dir"
            exit 1
        }
        return 0
    fi

    log_error "Cannot create directory and sudo not available: $dir"
    exit 1
}

install_binary() {
    local zip_file="$1"
    local install_dir="$2"
    local extracted_dir="$TEMP_DIR/extracted"
    local binary_path

    mkdir -p "$extracted_dir"
    unzip -q "$zip_file" -d "$extracted_dir"

    binary_path="$extracted_dir/$BINARY_NAME"
    if [ ! -f "$binary_path" ]; then
        # Handle binaries with platform suffix (sfs-darwin-arm64) or .exe suffix
        binary_path="$(find "$extracted_dir" \( -name "${BINARY_NAME}" -o -name "${BINARY_NAME}.exe" -o -name "${BINARY_NAME}-${OS}-${ARCH}" -o -name "${BINARY_NAME}-${OS}-${ARCH}.exe" \) -type f | head -n 1)"
        if [ -z "$binary_path" ]; then
            log_error "Binary '$BINARY_NAME' not found in archive"
            exit 1
        fi
    fi

    chmod +x "$binary_path"
    ensure_dir_exists "$install_dir"

    if [ -w "$install_dir" ]; then
        if has_cmd install; then
            install -m 0755 "$binary_path" "$install_dir/$BINARY_NAME"
        else
            cp "$binary_path" "$install_dir/$BINARY_NAME"
            chmod 0755 "$install_dir/$BINARY_NAME"
        fi
    elif ! is_termux && has_cmd sudo; then
        if has_cmd install; then
            sudo install -m 0755 "$binary_path" "$install_dir/$BINARY_NAME" || {
                log_error "sudo install failed: $install_dir/$BINARY_NAME"
                exit 1
            }
        else
            sudo cp "$binary_path" "$install_dir/$BINARY_NAME" || {
                log_error "sudo cp failed: $install_dir/$BINARY_NAME"
                exit 1
            }
            sudo chmod 0755 "$install_dir/$BINARY_NAME" || {
                log_error "sudo chmod failed: $install_dir/$BINARY_NAME"
                exit 1
            }
        fi
    else
        log_error "Directory not writable and cannot escalate: $install_dir"
        exit 1
    fi

    if [ -x "$install_dir/$BINARY_NAME" ]; then
        log_info "Installation complete: $install_dir/$BINARY_NAME"
        "$install_dir/$BINARY_NAME" --version 2>/dev/null || true
    else
        log_error "Installation verification failed"
        exit 1
    fi

    case ":$PATH:" in
        *":$install_dir:"*)
            ;;
        *)
            log_warn "$install_dir is not in PATH, please add it manually"
            ;;
    esac
}

main() {
    parse_args "$@"
    init_temp_dir

    require_download_tool
    require_extract_tool
    fetch_latest_version
    detect_platform

    INSTALL_PATH="$(determine_install_dir)"
    log_info "Installing ${BINARY_NAME} ${VERSION}"
    log_info "Install directory: $INSTALL_PATH"

    get_download_info
    log_info "Download URL: $DOWNLOAD_URL"

    local zip_file="$TEMP_DIR/$FILENAME"
    download_file "$DOWNLOAD_URL" "$zip_file"

    local expected_sha
    local sha_url="${DOWNLOAD_URL}.sha256"
    if expected_sha="$(get_sha256_hash "$sha_url")" && [ -n "$expected_sha" ]; then
        if ! verify_sha256 "$zip_file" "$expected_sha"; then
            prompt_continue_or_abort "SHA256 verification failed"
        fi
    else
        prompt_continue_or_abort "Could not fetch SHA256 checksum"
    fi

    install_binary "$zip_file" "$INSTALL_PATH"
}

main "$@"
