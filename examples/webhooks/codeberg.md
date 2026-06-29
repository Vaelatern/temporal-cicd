# Codeberg (Forgejo/Gitea) Webhook Deployment for Kickoff

## Endpoint
`POST /hooks/codeberg/{your-internal-repo-name}?token=YOUR_AUTH_TOKEN`

The `{repo}` provides the canonical internal name.

## Configuration in Codeberg / Forgejo / Gitea
1. Repo Settings > Webhooks > Add webhook (Gitea/Forgejo webhook)
2. Target URL: `https://your-kickoff.example.com/hooks/codeberg/{your-internal-repo-name}?token=the-token-from-keys.d`
3. Secret: set to the webhook shared secret (for `X-Gitea-Signature`)
4. Events: Push events
5. Use "Forgejo" or "Gitea" type if the option exists

## keys.d Configuration (auth token)
```
the-token:
  - ^POST /hooks/codeberg
  - ^KICKOFF /.*
```

## Webhook Shared Secrets
```yaml
# codeberg-secrets.yaml
- provider: codeberg
  key: "your-forgejo-webhook-secret"
  valid-repo: "owner/.*"
```

Falls back to `X-Hub-Signature*` headers if `X-Gitea-Signature` absent.

## Notes
- Payload similar to GitHub: `full_name` + `ref` (normalized, deleted pushes ignored).
- Query param `?token` for auth token; webhook secret configured independently in shared-secrets.
- SIGUSR1 reloads secrets.
