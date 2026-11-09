# syntax=docker/dockerfile:1
FROM nginx:alpine

# Build output of `gulp vscode-web-min-ci`, produced by the workflow:
# out/, extensions/, node_modules/, package.json, manifest.json, favicon.ico.
COPY vscode-web/ /usr/share/nginx/html/

# Authored shells (the upstream `web` target emits no HTML) and runtime config.
COPY index.html /usr/share/nginx/html/index.html
COPY agents.html /usr/share/nginx/html/agents.html
COPY config.json /usr/share/nginx/html/config.json

# MIT attribution for the redistributed Code - OSS build.
COPY LICENSE.txt /usr/share/nginx/html/LICENSE.txt
COPY ThirdPartyNotices.txt /usr/share/nginx/html/ThirdPartyNotices.txt

# Processed by /docker-entrypoint.d/20-envsubst-on-templates.sh at startup.
COPY nginx.conf.template /etc/nginx/templates/default.conf.template

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
	CMD wget -qO- http://127.0.0.1/healthz >/dev/null 2>&1 || exit 1
