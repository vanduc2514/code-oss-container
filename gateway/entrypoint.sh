#!/bin/sh
# Start ahp-acp-gateway with the mounted configuration file.
#
# The file is native ahp-acp-gateway configuration and is passed through
# unchanged. Nothing is parsed, rendered, or rewritten.
set -eu

CONFIG="${AHP_GATEWAY_CONFIG:-/etc/ahp/gateway.json}"

if [ ! -f "$CONFIG" ]; then
	echo "[entrypoint] gateway configuration not found: $CONFIG" >&2
	echo "[entrypoint] mount a gateway.json there, for example ./gateway.example.json" >&2
	exit 1
fi

echo "[entrypoint] using configuration: $CONFIG"

exec ahp-acp-gateway --config "$CONFIG"
