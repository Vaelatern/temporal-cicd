# GitHub Webhook Deployment for Kickoff

## Endpoint
`POST /hooks/github/{your-internal-repo-name}?token=YOUR_AUTH_TOKEN`

The `{repo}` segment in the path provides the canonical repository name used by the system (it overrides any `full_name` from the payload).

## Configuration in GitHub
1. Go to repo Settings > Webhooks > Add webhook
2. Payload URL: `https://your-kickoff.example.com/hooks/github/{your-internal-repo-name}?token=the-token-from-keys.d`
3. Content type: application/json
4. Secret: (recommended) the webhook shared secret value (configured separately via shared-secrets; see below). This enables `X-Hub-Signature-256` (or legacy `X-Hub-Signature`) verification.
5. Events: Just the push event (or send everything; handler ignores non-push)
6. Active: yes

## keys.d Configuration (auth token)
In your `keys.d/` yaml file (the token must permit the webhook path):
```
the-token:
  - ^POST /hooks/github
  - ^KICKOFF /.*
```

## Webhook Shared Secrets (signature verification)
Webhook secrets for HMAC verification are configured separately from auth tokens, via the `shared-secrets` directory (default: `../shared-secrets`, or set via `TCD_DIR_SHAREDSECRETS` / config `shared_secrets.dir`).

Place one or more `.yaml` files in that directory. Format is a list:

```yaml
# github-secrets.yaml
- provider: github
  key: "your-github-webhook-secret-value"
  valid-repo: "owner/.*"   # optional regexp; omit to match any repo for this provider

- key: "fallback-secret-for-any-provider"
```

- `provider` is optional (matches any if omitted)
- `valid-repo` is optional regexp (defaults to any)
- Secrets are selected by first match on provider + repo regexp.
- The secret is used only if a signature header is present from the provider.
- Changes are picked up on SIGUSR1 (or restart).

## Notes
- Signature verification uses constant-time HMAC-SHA256 compare.
- Payload: extracts `ref` (normalized by stripping `refs/heads/` or `refs/tags/`), ignores deleted pushes.
- `?token` supplies the auth token (from keys.d); the webhook `Secret` field is the separate shared secret.
- Test deliveries come from GitHub; use the "Recent Deliveries" view for debugging.
