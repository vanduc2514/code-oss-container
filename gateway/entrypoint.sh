#!/bin/sh
# Start ahp-acp-gateway with the mounted configuration file.
#
# The file is native ahp-acp-gateway configuration and is passed through
# unchanged. Credentials may arrive as environment variables or as one file per
# secret in AHP_SECRETS_DIR; an environment variable always wins.
set -eu

CONFIG="${AHP_GATEWAY_CONFIG:-/etc/ahp/gateway.json}"
SECRETS="${AHP_SECRETS_DIR:-/run/secrets}"

if [ ! -f "$CONFIG" ]; then
	echo "[entrypoint] gateway configuration not found: $CONFIG" >&2
	echo "[entrypoint] mount a gateway.json there, for example ./gateway.example.json" >&2
	exit 1
fi

# load_secret VAR file: export VAR from a file when the variable is empty.
load_secret() {
	var="$1"
	file="$SECRETS/$2"
	if [ -n "$(printenv "$var" 2>/dev/null || true)" ]; then
		return 0
	fi
	if [ ! -r "$file" ]; then
		return 0
	fi
	value="$(cat "$file")"
	if [ -n "$value" ]; then
		export "$var=$value"
		echo "[entrypoint] $var loaded from $file"
	fi
}

load_secret DEEPSEEK_API_KEY deepseek_api_key
load_secret COPILOT_GITHUB_TOKEN copilot_github_token
load_secret ANTHROPIC_API_KEY anthropic_api_key
load_secret OPENAI_API_KEY openai_api_key

echo "[entrypoint] using configuration: $CONFIG"

exec ahp-acp-gateway --config "$CONFIG"
