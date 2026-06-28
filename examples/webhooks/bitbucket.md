# Bitbucket Webhook Deployment for Kickoff

## Endpoint
`POST /hooks/bitbucket?token=YOUR_TOKEN`

## Configuration in Bitbucket Cloud
1. Repo Settings > Webhooks > Add webhook
2. URL: `https://your-kickoff.example.com/hooks/bitbucket/{your-internal-repo-name}?token=the-token-from-keys.d`
3. Secret: set to the same value as YOUR_TOKEN (enables X-Hub-Signature)
4. Triggers: Repository push (recommended)
5. Status: Active

## keys.d Configuration
```
the-token:
  - ^POST /hooks/bitbucket
  - ^KICKOFF /.*
```

## Notes
- Verifies with X-Hub-Signature (sha256=HMAC) when secret configured.
- Payload parsing handles nested push.changes[0].new.name for ref (branch name usually)
- repo full_name e.g. "owner/repo"
- ?token for auth + secret.
