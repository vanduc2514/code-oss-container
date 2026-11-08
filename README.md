# Code - OSS web container

A self-hosted web UI for **Code - OSS**, the open source project behind VS Code,
built from the unmodified upstream source at tag `1.145.2`
(commit `c4453f8ea430795f941c1b2ff6c9321135a74a02`) and served by nginx.

This repository contains **only** the container layer: a Dockerfile, an nginx
configuration, two HTML shells, and the CI that builds and publishes the image.
Upstream source is checked out at build time and is never copied into this
repository or modified.

- `/` — the Code - OSS web workbench (`vs/workbench/workbench.web.main.internal`)
- `/agents` — the Agents Window (`vs/sessions/sessions.web.main.internal`)

## Why the shells exist

The upstream `web` build target (`npm run gulp vscode-web-min-ci`) emits
JavaScript and CSS only — no HTML. Microsoft's own pages at vscode.dev are
external shells. This project therefore authors two small pages that call the
`create()` entry points, following the same structure as upstream's own
development shells.

Both shells:

- set `globalThis._VSCODE_FILE_ROOT` and load `out/nls.messages.js` in classic
  scripts before the module graph, which upstream requires;
- pass a `productConfiguration` using Code - OSS naming, `enableTelemetry: false`,
  and the Open VSX gallery (never a Microsoft service);
- load `extensions/theme-defaults` and `extensions/theme-seti` as additional
  builtin extensions so the UI is themed;
- fetch `/config.json` at boot and use it for `chat.remoteAgentHosts`.

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
  ghcr.io/vanduc2514/code-oss-container:1.145.2
```

Open <http://localhost:9000/> and <http://localhost:9000/agents>.
Health endpoint: `GET /healthz`.

## Agent hosts (AHP)

The Agents Window talks to Agent Host Protocol servers over WebSocket. The UI is
host-agnostic: any AHP host that advertises models works, and hosts are supplied
at runtime, not baked into the image.

```jsonc
// /usr/share/nginx/html/config.json
{
  "remoteAgentHosts": [
    { "name": "my-host", "address": "ws://localhost:3013", "connectionToken": "optional" }
  ]
}
```

Mount your own file over the default:

```bash
docker run --rm -d --name code-oss -p 9000:80 \
  -v "$PWD/config.json:/usr/share/nginx/html/config.json:ro" \
  ghcr.io/vanduc2514/code-oss-container:1.145.2
```

The agent host is dialled by the **browser**, not by the container: the
workbench and its extension host run in the page, so `localhost` means the
machine running the browser. Use a host the browser can reach; a remote browser
needs the port published or a reverse proxy of its own.

`/config.json` is served `Cache-Control: no-store`, so a page reload picks up
changes without rebuilding or restarting.

Notes:

- A Copilot-backed host (`copilotcli`) requires a GitHub credential *from the
  browser* to start a session, regardless of the host's own sign-in. Non-Copilot
  hosts do not.
- `chat.allowAnonymousAccess` is enabled in both shells, which keeps the
  signed-out UI from demanding a sign-in before a harness is selectable.

## Build

The workflow mirrors Microsoft's own web pipeline
(`build/azure-pipelines/web/product-build-web.yml`):

```bash
npm ci
npm run gulp core-ci
npm run gulp vscode-web-min-ci
```

`vscode-web-min-ci` writes to `../vscode-web` relative to the repository root.
The workflow moves that artifact, caches it keyed on tag + commit, verifies the
required bundles, and builds the image with Docker Buildx.

Two build-time details:

- Ubuntu 22.04 ships GCC 11.4, which cannot parse the Electron 43 V8 headers
  when `USING_V8_SHARED` and the V8 deprecation-warning macros are both defined.
  `npm ci` therefore runs with
  `CFLAGS`/`CXXFLAGS='-UV8_DEPRECATION_WARNINGS -UV8_IMMINENT_DEPRECATION_WARNINGS'`.
  This only silences deprecated-API warnings in native module builds.
- The workflow asserts the checked out commit equals the pinned tag **and** that
  `git status --porcelain` is empty, so a modified upstream tree fails the build.

A `smoke` job pulls the pushed image and asserts the served pages, bundle sizes,
content types, `/healthz`, attribution files, and the absence of any
`microsoft.com`, `vscode.dev`, `vscode-cdn.net`, `marketplace.visualstudio.com`
or `aka.ms` reference.

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
