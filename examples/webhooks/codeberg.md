# Codeberg (Forgejo/Gitea) Webhook Deployment for Kickoff

## Endpoint
`POST /hooks/codeberg?token=YOUR_TOKEN`

## Configuration in Codeberg / Forgejo / Gitea
1. Repo Settings > Webhooks > Add webhook (or Add Gitea webhook)
2. Target URL: `https://your-kickoff.example.com/hooks/codeberg/{your-internal-repo-name}?token=the-token-from-keys.d`
3. Secret: set to same as YOUR_TOKEN (for X-Gitea-Signature)
4. Events: Push events
5. Use "Forgejo" or "Gitea" webhook type if options

## keys.d Configuration
```
the-token:
  - ^POST /hooks/codeberg
  - ^KICKOFF /.*
```

## Notes
- Signature via X-Gitea-Signature (HMAC-SHA256) or fallback X-Hub-*
- Similar payload to GitHub, full_name and ref.
- ?token query param supported for webhook URL as researched (all providers allow query params in target URL).
- Properly verifies origin provider signature when secret header present.
