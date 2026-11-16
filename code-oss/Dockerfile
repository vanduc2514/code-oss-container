# syntax=docker/dockerfile:1

# ---------------------------------------------------------------------------
# Compiles the startup helper that turns config.json into nginx proxy routes.
# Cross-compiled for the target architecture, so multi-arch builds need no
# emulation for this stage.
# ---------------------------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS routes
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY build/nginx-routes/ ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
      go build -trimpath -ldflags='-s -w' -o /out/nginx-routes .

# ---------------------------------------------------------------------------
# Build stage: clone the pinned upstream release, rewrite product.json, and
# run upstream's own web pipeline. Only the compiled web bundle leaves this
# stage; nothing is installed into the runtime image from here.
# ---------------------------------------------------------------------------
FROM --platform=$BUILDPLATFORM node:24-bookworm AS build

ARG VSCODE_TAG=1.145.2
ARG VSCODE_COMMIT=c4453f8ea430795f941c1b2ff6c9321135a74a02

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      build-essential \
      git \
      libkrb5-dev \
      libsecret-1-dev \
      libx11-dev \
      libxkbfile-dev \
      pkg-config \
      python3 \
 && rm -rf /var/lib/apt/lists/*

RUN git clone --depth 1 --branch "$VSCODE_TAG" \
      https://github.com/microsoft/vscode.git /src/vscode

WORKDIR /src/vscode

RUN actual="$(git rev-parse HEAD)" \
 && if [ "$actual" != "$VSCODE_COMMIT" ]; then \
      echo "checked out $actual, expected $VSCODE_COMMIT" >&2; \
      exit 1; \
    fi

COPY build/patch-product.mjs /tmp/patch-product.mjs
RUN node /tmp/patch-product.mjs /src/vscode/product.json

ENV ELECTRON_SKIP_BINARY_DOWNLOAD=1 \
    PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 \
    CFLAGS="-UV8_DEPRECATION_WARNINGS -UV8_IMMINENT_DEPRECATION_WARNINGS" \
    CXXFLAGS="-UV8_DEPRECATION_WARNINGS -UV8_IMMINENT_DEPRECATION_WARNINGS"

# Upstream writes the web bundle to the checkout's parent directory.
RUN npm ci \
 && npm run gulp core-ci \
 && npm run gulp vscode-web-min-ci \
 && npm cache clean --force

# ---------------------------------------------------------------------------
# Runtime stage: static files plus nginx. Agent hosts and proxy targets come
# from the mounted config.json at container start; nothing depends on compose.
# ---------------------------------------------------------------------------
FROM nginx:alpine AS runtime

# The shells report the same build identity upstream's own web shells do; link
# it to the pin instead of repeating it in two entry-point modules.
ARG VSCODE_TAG
ARG VSCODE_COMMIT

COPY --from=build /src/vscode-web/ /usr/share/nginx/html/
COPY --from=build /src/vscode/LICENSE.txt /src/vscode/ThirdPartyNotices.txt /usr/share/nginx/html/

COPY runtime/index.html runtime/agents.html \
     runtime/shell.js runtime/index.js runtime/agents.js /usr/share/nginx/html/
RUN test -n "$VSCODE_TAG" && test -n "$VSCODE_COMMIT" \
 && printf 'export const version = "%s";\nexport const commit = "%s";\n' \
      "$VSCODE_TAG" "$VSCODE_COMMIT" > /usr/share/nginx/html/version.js

COPY runtime/nginx.conf.template /etc/nginx/templates/default.conf.template
COPY --from=routes /out/nginx-routes /usr/local/bin/nginx-routes
COPY runtime/25-nginx-routes.sh /docker-entrypoint.d/25-nginx-routes.sh
RUN chmod +x /docker-entrypoint.d/25-nginx-routes.sh && mkdir -p /etc/nginx/ahp

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
	CMD wget -qO- http://127.0.0.1/healthz >/dev/null 2>&1 || exit 1
