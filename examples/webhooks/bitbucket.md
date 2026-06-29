# Bitbucket Webhook Deployment for Kickoff

## Endpoint
`POST /hooks/bitbucket/{your-internal-repo-name}?token=YOUR_AUTH_TOKEN`

The `{repo}` segment provides the canonical name (overrides payload `full_name`).

## Configuration in Bitbucket Cloud
1. Repo Settings > Webhooks > Add webhook
2. URL: `https://your-kickoff.example.com/hooks/bitbucket/{your-internal-repo-name}?token=the-token-from-keys.d`
3. Secret: set to the webhook shared secret (enables `X-Hub-Signature` / `X-Hub-Signature-256`)
4. Triggers: Repository push (recommended)
5. Status: Active

## keys.d Configuration (auth token)
```
the-token:
  - ^POST /hooks/bitbucket
  - ^KICKOFF /.*
```

## Webhook Shared Secrets
Via `shared-secrets/` directory (yaml list):

```yaml
- provider: bitbucket
  key: "bitbucket-webhook-secret"
  valid-repo: "owner/.*"
```

Only signature verification (no token header for Bitbucket in this implementation).

## Notes
- Parses `push.changes[0].new.name` for the ref (usually branch name; normalized).
- Secret only used for signature if header present.
- `?token` supplies auth.
