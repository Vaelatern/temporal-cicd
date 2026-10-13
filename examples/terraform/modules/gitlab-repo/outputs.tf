output "name" {
  value = lrcicd_repo.this.name
}

output "auth_token" {
  value     = local.token
  sensitive = true
}

output "webhook_secret" {
  value     = random_password.webhook_secret.result
  sensitive = true
}

output "webhook_url" {
  value = gitlab_project_hook.kickoff.url
}
