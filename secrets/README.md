# Credential files

One file per secret, mounted read-only into the gateway at `/run/secrets`. The
entrypoint exports a variable from a file only when that variable is empty, so
`.env` still wins when both are set.

| File | Variable | Used by |
|---|---|---|
| `deepseek_api_key` | `DEEPSEEK_API_KEY` | pi, Copilot BYOK |
| `copilot_github_token` | `COPILOT_GITHUB_TOKEN` | Copilot |
| `anthropic_api_key` | `ANTHROPIC_API_KEY` | Claude |
| `openai_api_key` | `OPENAI_API_KEY` | Codex |
| `antigravity-oauth-token` | none, mounted as a file | Antigravity |

Paste the value with no surrounding quotes and no trailing whitespace:

```bash
printf '%s' "$SECRET" > secrets/deepseek_api_key
chmod 600 secrets/deepseek_api_key
```

`antigravity-oauth-token` is the file the Antigravity CLI writes when you log in,
usually at `~/.gemini/antigravity-cli/antigravity-oauth-token`. Copy it here and
the container reads it in place; the bind mount is writable so the token can
refresh. Deleting the file lets Antigravity fall back to an interactive login
inside the container.

Do not commit anything in this directory. Only this README is tracked.
