# Code - OSS web container

A self-hosted web UI for **Code - OSS**, the open source project behind VS Code,
built from upstream source at tag `1.145.2`
(commit `c4453f8ea430795f941c1b2ff6c9321135a74a02`) and served by nginx.

The image lives under [`code-oss/`](code-oss) and is built by one multi-stage
Dockerfile: the build stage clones and compiles upstream, the runtime stage
serves the result with nginx and contains no toolchain or upstream checkout.
Build-time code lives in `code-oss/build/`, runtime files in `code-oss/runtime/`.
The repository root adds `compose.yaml`, an example deployment with a private
AHP gateway. Before compiling,
[`patch-product.mjs`](code-oss/build/patch-product.mjs) rewrites the chat-agent
endpoints in the checked-out `product.json`; see [Build](#build).

- `/` — the Code - OSS web workbench (`vs/workbench/workbench.web.main.internal`)
- `/agents` — the Agents Window (`vs/sessions/sessions.web.main.internal`)

## Repository layout

```
code-oss/
  Dockerfile              multi-stage: compile upstream, then nginx runtime
  build/
    patch-product.mjs     build-time product.json rewrite
    nginx-routes/         Go helper: config.json -> nginx proxy routes
  runtime/
    index.html            workbench shell
    agents.html           Agents Window shell
    index.js              workbench entry point module
    agents.js             Agents Window entry point module
    shell.js              helpers shared by both entry points
    nginx.conf.template   rendered by envsubst at container start
    25-nginx-routes.sh    entrypoint wrapper that runs the Go helper
                          (version.js is generated into the image at build time)
gateway/                  AHP gateway image: pi, Copilot, BYOK providers
compose.yaml              example ui + gateway deployment
config.example.json       example agent-host list, mounted at /config.json
.env.example              example stack configuration
.github/workflows/container.yml
README.md
```

## Why the shells exist

The upstream `web` build target (`npm run gulp vscode-web-min-ci`) emits
JavaScript and CSS only — no HTML. Microsoft's own pages at vscode.dev are
external shells. This project therefore authors two small pages plus their entry
point modules, following the same structure as upstream's own development
shells.

Both shells:

- load `out/nls.messages.js` as a classic script, then set
  `globalThis._VSCODE_FILE_ROOT` from their entry point module before importing
  the workbench bundle, which upstream requires;
- keep their boot logic in separate entry point modules, so the large
  `productConfiguration` object reads as JavaScript instead of an inline script;
- pass a `productConfiguration` using Code - OSS naming, `enableTelemetry: false`,
  and the Open VSX gallery (never a Microsoft service);
- load `extensions/theme-defaults` and `extensions/theme-seti` as additional
  builtin extensions so the UI is themed;
- fetch `/config.json` at boot and use it for `chat.remoteAgentHosts`.

`agents.html` also sets the `skip-sessions-welcome` query parameter, the same
signed-out escape hatch upstream's `scripts/code-sessions-web.js --skip-welcome`
uses, so the window opens without a GitHub sign-in.

## Image

```
ghcr.io/vanduc2514/code-oss-container:1.145.2
ghcr.io/vanduc2514/code-oss-container:latest
ghcr.io/vanduc2514/code-oss-container:c4453f8ea430795f941c1b2ff6c9321135a74a02
```

`linux/amd64` and `linux/arm64`, published as one manifest.

## Run

```bash
docker run --rm -d --name code-oss -p 9000:80 \
  -v "$PWD/config.json:/usr/share/nginx/html/config.json:ro" \
  ghcr.io/vanduc2514/code-oss-container:1.145.2
```

Open <http://localhost:9000/> and <http://localhost:9000/agents>.
Health endpoint: `GET /healthz`.

Agent hosts come only from the mounted `config.json`; without the mount the UI
starts with no hosts. See [Agent hosts (AHP)](#agent-hosts-ahp) for the file
format.

## Full stack: Agents UI + AHP gateway

`compose.yaml` is an example deployment of the two images in this repository:
the Code - OSS image and the `gateway/` image, which hosts `pi`, the GitHub
Copilot CLI, and the Copilot BYOK providers. Both images are generic; compose
only wires the environment variables. Only the UI is published, and the gateway
is reachable inside the deployment network.

Build the UI image locally, or use the published one:

```bash
docker build -t code-oss-container:local ./code-oss
```

Then deploy:

```bash
cp config.example.json config.json   # edit address and connectionToken
cp .env.example .env                 # set AHP_CONFIG_FILE, AHP_AUTH_TOKEN,
                                     # DEEPSEEK_API_KEY, COPILOT_GITHUB_TOKEN
docker compose up -d
```

- `ui` serves `/` and `/agents` on <http://localhost:9000> and proxies
  `ws://<host>/ahp` to the `upstream` in `config.json`. `UI_IMAGE` names the
  Code - OSS image (for example `code-oss-container:local` after the build).
- `gateway` renders `gateway.json` from `AHP_*` variables at startup and keeps
  its SQLite database, pi state, and Copilot login in named volumes.
- `config.json` is mounted read-only at `/usr/share/nginx/html/config.json` and
  points the browser at the proxy. Its `connectionToken` must match
  `AHP_AUTH_TOKEN`. The image ships no default file.

Traefik (or any TLS proxy) terminates TLS and forwards HTTP and WebSocket
upgrades to the published UI port. Put the public address in the mounted
`config.json`, for example `wss://agents.example.com/ahp`.

The gateway requires `AHP_AUTH_TOKEN` because it binds a non-loopback address.
The same value goes into the browser's `config.json` as `connectionToken` and is
sent as the `?token=` query parameter, so protect the UI port with the proxy or
network policy.

Provider, secret, and Copilot login details: [gateway/README.md](gateway/README.md).

Standalone runs of the image do not include the gateway; they use whatever AHP
hosts you mount in `config.json`.

## Agent hosts (AHP)

The Agents Window talks to Agent Host Protocol servers over WebSocket. The UI is
host-agnostic: any AHP host that advertises models works, and hosts are supplied
at runtime, not baked into the image.

```jsonc
// /usr/share/nginx/html/config.json
{
  "remoteAgentHosts": [
    { "name": "my-host", "address": "wss://ui.example.com/ahp/my-host", "connectionToken": "optional" }
  ]
}
```

Mount your own file:

```bash
docker run --rm -d --name code-oss -p 9000:80 \
  -v "$PWD/config.json:/usr/share/nginx/html/config.json:ro" \
  ghcr.io/vanduc2514/code-oss-container:1.145.2
```

The image ships no `config.json`. Without the mount, `/config.json` returns 404
and both shells start with no agent hosts.

`/config.json` is served `Cache-Control: no-store`, so a page reload picks up
changes without rebuilding or restarting.

### Proxy-only fields

An entry can instead name a private upstream. The container then proxies that
host, and the browser only ever connects to this origin:

```jsonc
// /usr/share/nginx/html/config.json
{
  "remoteAgentHosts": [
    { "name": "AHP Gateway", "path": "/ahp", "upstream": "http://gateway:8765/ahp", "connectionToken": "..." },
    { "name": "team-a",      "path": "/ahp/team-a", "upstream": "http://gateway-a.private:8765/ahp" },
    { "name": "external",    "address": "wss://agents.example.com/ahp" }
  ]
}
```

| Field | Meaning |
|---|---|
| `name` | Display name |
| `path` | Same-origin path, used as the browser address when `address` is absent |
| `upstream` | Private URL the container proxies to; absent means the browser dials `address` directly |
| `address` | Explicit browser-facing URL; wins over `path` |
| `connectionToken` | Token sent as `?token=` |

At container start, the [`nginx-routes`](code-oss/build/nginx-routes) helper
reads the file and writes one self-contained nginx location per entry that has
an `upstream`, using `path` or a slug of `name`:

```nginx
location = /ahp/team-a {
	access_log off;
	proxy_pass http://gateway-a.private:8765/ahp;   # /ahp/team-a becomes /ahp
	proxy_http_version 1.1;
	proxy_set_header Upgrade $http_upgrade;
	# ...upgrade headers and long read/send timeouts
}
```

So adding a host is a `config.json` edit plus a restart; no nginx files to
write. Invalid paths or upstreams abort startup with an error rather than
producing a broken configuration.

The browser never resolves a private name, and `shell.js` passes only `name`,
`address`, and `connectionToken` to the workbench. The mounted file itself is
still served over HTTP, so anyone who can read `/config.json` can see the
`upstream` values; they are harmless unless reachable from outside the private
network.

The ingress in front of this container must forward WebSocket upgrades on these
paths, the same as for `/ahp`.

Notes:

- A Copilot-backed host (`copilotcli`) requires a GitHub credential *from the
  browser* to start a session, regardless of the host's own sign-in. Non-Copilot
  hosts do not.
- `chat.allowAnonymousAccess` is enabled in both shells, which keeps the
  signed-out UI from demanding a sign-in before a harness is selectable.

## Build

One multi-stage Dockerfile builds everything:

```bash
docker build -t code-oss-container ./code-oss
```

The `routes` stage cross-compiles the small Go helper that generates the `/ahp`
proxy locations at container start, for the target architecture. Its tests run
without Docker:

```bash
cd code-oss/build/nginx-routes && go test ./...
```

The `build` stage runs on the host architecture (the web bundle is platform
independent) and mirrors Microsoft's own web pipeline
(`build/azure-pipelines/web/product-build-web.yml`):

1. clones `microsoft/vscode` at the `VSCODE_TAG` build arg and asserts the
   checked-out commit equals `VSCODE_COMMIT`;
2. rewrites the chat-agent endpoints in `product.json` with
   [`patch-product.mjs`](code-oss/build/patch-product.mjs), because several
   workbench code paths read the build-time product object instead of
   `IProductService` and would otherwise still call `aka.ms` and
   `api.github.com` when chat setup runs;
3. runs `npm ci`, `npm run gulp core-ci`, `npm run gulp vscode-web-min-ci`.

The `runtime` stage is `nginx:alpine` plus the compiled bundle, the shells, and
upstream's attribution files. It also writes `version.js` from the same
`VSCODE_TAG`/`VSCODE_COMMIT` build args, which both shells import as their
`version` and `commit`, so the pin is declared once. No Node.js, compiler, or upstream checkout ships
in the final image. It reads no runtime environment variables: agent hosts and
proxy targets come entirely from the mounted `config.json`.

The `VSCODE_TAG` and `VSCODE_COMMIT` ARGs in the Dockerfile are the single
version pin, and CI reads them to tag the published image. To build another
release:

```bash
docker build \
  --build-arg VSCODE_TAG=1.146.0 \
  --build-arg VSCODE_COMMIT=<sha> \
  -t code-oss-container:1.146.0 ./code-oss
```

A cold build takes roughly 45-60 minutes and 20 GiB of disk. BuildKit caches the
expensive stage, and CI persists that cache with `type=gha`.

One build-time detail: the build stage defines
`CFLAGS`/`CXXFLAGS='-UV8_DEPRECATION_WARNINGS -UV8_IMMINENT_DEPRECATION_WARNINGS'`
because some GCC versions cannot parse the Electron 43 V8 headers while those
macros are defined. This only silences deprecated-API warnings in native module
builds.

### CI

A `test` job runs `go test ./...` for the route helper with the Go version from
its `go.mod`, and the image build waits for it.

The workflow builds and pushes the multi-arch image directly from `code-oss/`
with Docker Buildx, passing the pinned version as build args and caching the
build stage in GitHub Actions. A `smoke` job pulls the pushed image and asserts
the served pages, bundle sizes, content types, `/healthz`, attribution files,
and the absence of any `microsoft.com`, `vscode.dev`, `vscode-cdn.net`,
`marketplace.visualstudio.com` or `aka.ms` reference.

## Licensing and trademarks

- Upstream Code - OSS is MIT licensed. The image ships upstream's `LICENSE.txt`
  and `ThirdPartyNotices.txt` at the web root for attribution.
- The product identity here is **Code - OSS**, not Visual Studio Code. The
  shells use Code - OSS names, disable telemetry, and point extensions at Open
  VSX rather than any Microsoft gallery or service.
- No Microsoft logo, icon, or product name is used, and nothing is presented as
  an official Microsoft build.
- VS Code and Visual Studio are trademarks of Microsoft. This project is not
  affiliated with or endorsed by Microsoft.
