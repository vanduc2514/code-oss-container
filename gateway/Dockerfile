# syntax=docker/dockerfile:1
# ahp-acp-gateway only. No agent is baked into the image: providers are
# installed into the /opt/agents volume by the compose init service and started
# from PATH, so agent versions and credentials are deployment concerns.

FROM node:24-bookworm-slim

ARG GATEWAY_VERSION=0.2.2

# git/ripgrep are used by agent tooling; tini gives clean signal handling.
RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      ca-certificates \
      git \
      ripgrep \
      bash \
      procps \
      tini \
 && rm -rf /var/lib/apt/lists/*

RUN npm install --global --no-audit --no-fund "ahp-acp-gateway@${GATEWAY_VERSION}" \
 && npm cache clean --force

# These directories must exist and be node-owned so named volumes mounted over
# them inherit the right ownership on first use.
RUN mkdir -p /data /workspace /etc/ahp \
      /home/node/.copilot /home/node/.pi \
      /home/node/.gemini/antigravity-cli \
 && chown -R node:node /data /workspace /home/node/.copilot /home/node/.pi \
      /home/node/.gemini

COPY --chmod=755 entrypoint.sh /usr/local/bin/ahp-entrypoint

USER node

# Agents are installed into the agents volume by the compose init service, not
# into this image. Their binaries must come first on PATH.
ENV HOME=/home/node \
    NODE_ENV=production \
    AHP_GATEWAY_CONFIG=/etc/ahp/gateway.json \
    PATH=/opt/agents/bin:/opt/agents/node_modules/.bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

EXPOSE 8765

# Reads the port from the mounted configuration, so a different listener port
# needs no change here.
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD node -e "const fs=require('fs');const c=JSON.parse(fs.readFileSync(process.env.AHP_GATEWAY_CONFIG,'utf8'));fetch('http://127.0.0.1:'+c.listener.port+'/healthz').then(r=>process.exit(r.ok?0:1)).catch(()=>process.exit(1))"

ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/ahp-entrypoint"]
