# Code - OSS web container

This repository runs a self-hosted copy of the Code - OSS web IDE and its Agents
Window, together with the piece that makes the window usable: an Agent Host
Protocol (AHP) gateway that starts coding agents and streams them to the browser.
Two images do the work: one serves the editor as static files, the other runs the
gateway that owns sessions and the agent processes. Compose wires them together as
a working starting point rather than a hardened production setup.

## Quickstart

```bash
docker compose up -d
```

Open <http://localhost:9000> for the editor and <http://localhost:9000/agents>
for the Agents Window. The first run builds the gateway image, installs pi and
GitHub Copilot into a volume, and starts the stack. Ports 9000 and 8765 must be
free.

Both images are published: `ghcr.io/vanduc2514/code-oss-container` for the
editor and `ghcr.io/vanduc2514/ahp-gateway` for the gateway. On a homelab or any
fresh host, pull them before starting:

```bash
docker compose pull
docker compose up -d
```

If the gateway package is still private, log in first with a token that can read
packages (`docker login ghcr.io`), or make the package public in its GitHub
settings.

To build the editor locally instead:

```bash
docker build -t code-oss-container:local ./code-oss
UI_IMAGE=code-oss-container:local docker compose up -d
```

The gateway follows the same pattern. Compose builds it when the image is not
present locally, and `GATEWAY_IMAGE=ahp-stack-gateway:local docker compose up
--build` keeps a local build instead of the published one.

Everything starts without credentials. Agents appear in the picker, and a
session fails only when it reaches a model. To use them, copy `.env.example` to
`.env` and fill in what you have:

```dotenv
DEEPSEEK_API_KEY=sk-...
COPILOT_GITHUB_TOKEN=github_pat_...
```

The gateway and the browser share a token so the browser can authenticate. The
example uses `example-token-change-me`, which lives in `gateway.example.json` and
`agent-hosts.example.json`. Change both together before the stack is reachable
from anywhere but your machine. The commented variables in `.env.example` cover
ports, file paths, and image pins and are optional.

## How the pieces talk

Only the port 9000 container is published. It serves the editor files and hands
the browser a `/config.json` listing agent hosts. When a host entry names a
private upstream, that container also proxies the WebSocket, so the browser never
resolves an internal name.

The gateway listens on 8765, keeps sessions in SQLite, and spawns one process per
agent session. It speaks ACP over stdin and stdout to each agent, and AHP over
WebSocket to the browser. Model calls happen inside the agent process, so keys
stay on the gateway host and never reach the page.

A short-lived init service installs the agents, so the gateway image holds only
the gateway and Node. Agent versions and their large binaries stay out of the
image, and you change them by editing `.env` and restarting.

## Deployment architecture

A typical deployment is one Linux host with Docker, a TLS proxy such as Traefik
in front, and browsers somewhere else. Traefik terminates TLS and forwards HTTP
and WebSocket upgrades to the UI container on port 9000. The gateway stays on the
private compose network or, if you keep it published, on 8765 without TLS of its
own. Removing the `ports:` entry for the gateway is enough to keep it off the
host network, but then the UI has to proxy to it, which needs a UI image built
from this repository. The published tags predate that proxy helper.

The address in the agent host file is what the browser dials, so it has to match
how you expose the stack: `ws://localhost:8765/ahp` for a local run,
`ws://<host>:8765/ahp` when the gateway port is published, or
`wss://<domain>/ahp` behind the TLS proxy.

State lives in named volumes. `gateway-data` holds sessions and transcripts,
`gateway-workspace` is the directory agents read and write, `agents` holds the
installed agent binaries, and `copilot-auth` and `pi-home` keep logins so a
recreated container does not ask again. Credentials stay in `.env` or in a
secrets manager; nothing in this repository expects a key in an image or in a
config file.

## Customizing

The gateway reads one file, mounted at `/etc/ahp/gateway.json`. It is native
`ahp-acp-gateway` configuration and is passed through untouched, so any field that
gateway supports works here. Copy the example and point the stack at your copy:

```bash
cp gateway.example.json gateway.json
# set AHP_GATEWAY_CONFIG=./gateway.json in .env
docker compose up -d gateway
```

Each provider entry says what to run, in which directory, with which environment
and timeouts. The example enables GitHub Copilot, two Copilot sessions pointed at
DeepSeek, pi, and Antigravity. Adding a model to Copilot means copying a BYOK
entry and changing the base URL, model, and key command. Adding a different agent
means adding an entry whose command names a binary installed in the `agents`
volume. Disabled entries at the bottom of the file are the gateway's built-in
providers; delete the `enabled: false` line to use one, keeping in mind that its
package then has to be installed.

Check a change before restarting the container:

```bash
docker compose exec gateway \
  ahp-acp-gateway --config /etc/ahp/gateway.json --print-config
```

The init service installs pinned versions of `pi-acp`, `pi`, and the Copilot CLI.
Versions come from `.env`. Antigravity is the exception: its CLI is a native
binary from Google, not an npm package, so put a Linux build on the host and set
`AGY_BIN` to its path. Without it the service logs that Antigravity is
unavailable and everything else still works.

Logins persist in volumes. If you prefer an interactive login over an API token:

```bash
docker compose exec gateway copilot login --device-code
docker compose exec gateway pi
```

The agent host file is separate from the gateway file and only concerns the
browser. An entry with `address` connects directly. An entry with `path` and
`upstream` makes the UI container proxy to a private host, and the generated
nginx route uses `path` or a slug of `name`. Tokens go in `connectionToken` and
must match `auth.token` in the gateway file.

## Building the editor image

```bash
docker build -t code-oss-container ./code-oss
```

The build clones `microsoft/vscode` at the tag and commit pinned in the
Dockerfile, asserts the checkout matches, adjusts the chat endpoints in
`product.json`, and runs upstream's web build. The runtime stage is nginx with
the compiled bundle and no toolchain. A cold build takes about an hour and needs
around 20 GiB of disk.

CI builds both images for `linux/amd64` and `linux/arm64` and pushes them to the
GitHub registry, the editor from `container.yml` and the gateway from
`gateway.yml`. The gateway version comes from the ARG in its Dockerfile, so
bumping that value publishes `0.2.3` alongside `latest`.

## Licensing and trademarks

Upstream Code - OSS is MIT licensed, and the image ships its `LICENSE.txt` and
`ThirdPartyNotices.txt`. The product identity here is Code - OSS: telemetry is
off, extensions come from Open VSX, and no Microsoft service is contacted. VS
Code and Visual Studio are trademarks of Microsoft, and this project is not
affiliated with or endorsed by Microsoft.
