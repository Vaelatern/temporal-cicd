# GitLab Webhook Deployment for Kickoff

## Endpoint
`POST /hooks/gitlab/{your-internal-repo-name}?token=YOUR_AUTH_TOKEN`

The `{repo}` segment in the path provides the canonical repository name used by the system (it overrides any `path_with_namespace` from the payload).

## Configuration in GitLab
1. Go to repo Settings > Webhooks > Add new webhook
2. URL: `https://your-kickoff.example.com/hooks/gitlab/{your-internal-repo-name}?token=the-token-from-keys.d`
3. Secret Token: (or Signing secret) set to your webhook shared secret value (see below)
4. Trigger: Push events (and optionally others; handler filters non-push)
5. Enable SSL verification if applicable

## keys.d Configuration (auth token)
```
the-token:
  - ^POST /hooks/gitlab
  - ^KICKOFF /.*
```

## Webhook Shared Secrets (signature verification)
Configure via the `shared-secrets` directory (`.yaml` files, list format):

```yaml
# gitlab-secrets.yaml
- provider: gitlab
  key: "your-gitlab-webhook-secret"
  valid-repo: "group/subgroup/.*"
```

GitLab supports either:
- Exact match on `X-Gitlab-Token` header against the secret, **or**
- HMAC verification via `X-Gitlab-Signature` (or `webhook-signature`) using the secret.

The first matching secret (by provider + repo regexp) is used.

## Notes
- Ref normalized (strips `refs/heads/` / `refs/tags/`).
- `?token` for auth token; webhook secret is independent.
- Reload secrets with SIGUSR1.
