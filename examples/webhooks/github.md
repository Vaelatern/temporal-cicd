# GitHub Webhook Deployment for Kickoff

## Endpoint
`POST /hooks/github?token=YOUR_TOKEN`

## Configuration in GitHub
1. Go to repo Settings > Webhooks > Add webhook
2. Payload URL: `https://your-kickoff.example.com/hooks/github/{your-internal-repo-name}?token=the-token-from-keys.d`
3. Content type: application/json
4. Secret: (optional but recommended) Set to the SAME value as YOUR_TOKEN (used for X-Hub-Signature-256 verification)
5. Events: Just the push event (or send me everything, handler ignores non-push)
6. Active: yes

## keys.d Configuration
In your keys.d/ yaml file (token must allow the path):
```
the-token:
  - ^POST /hooks/github
  - ^KICKOFF /.*
```

## Notes
- Uses X-Hub-Signature-256 (or legacy sha1) for origin auth when secret set.
- Repository uses full_name e.g. "owner/repo"
- Ref normalized (strips refs/heads/ or tags/)
- Supports ?token for both auth and as webhook secret.

Test with: curl -X POST ... but use real delivery from GitHub.
