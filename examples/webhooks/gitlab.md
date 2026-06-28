# GitLab Webhook Deployment for Kickoff

## Endpoint
`POST /hooks/gitlab?token=YOUR_TOKEN`

## Configuration in GitLab
1. Go to repo Settings > Webhooks > Add new webhook
2. URL: `https://your-kickoff.example.com/hooks/gitlab/{your-internal-repo-name}?token=the-token-from-keys.d`
3. Secret Token: (legacy) or Signing token: set to same as YOUR_TOKEN
4. Trigger: Push events (and optionally others, handler filters)
5. Enable SSL verification if applicable

## keys.d Configuration
```
the-token:
  - ^POST /hooks/gitlab
  - ^KICKOFF /.*
```

## Notes
- Supports X-Gitlab-Token exact match (legacy) or HMAC via X-Gitlab-Signature / webhook-signature using token as secret.
- repo from path_with_namespace e.g. "group/subgroup/repo"
- Ref normalized.
- ?token passes the permission token and secret.
