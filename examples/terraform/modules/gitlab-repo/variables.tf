variable "name" {
  type        = string
  description = "Internal lrcicd repo alias (path segment in /sync and /hooks)."
}

variable "git_url" {
  type        = string
  description = "Upstream SSH git URL, e.g. git@gitlab.com:group/repo.git"
}

variable "ssh_reading_private_key" {
  type        = string
  sensitive   = true
  description = "SSH private key the cache uses to fetch the upstream."
}

variable "gitlab_project" {
  type        = string
  description = "GitLab project id or path for the project hook."
}

variable "kickoff_url" {
  type        = string
  description = "Public base URL of the kickoff service."
}

variable "nomad_keys_d_prefix" {
  type        = string
  default     = "temporal-cicd/keys-d"
  description = "Nomad variable path prefix for keys.d auth tokens."
}

variable "nomad_shared_secrets_prefix" {
  type        = string
  default     = "temporal-cicd/shared-secrets"
  description = "Nomad variable path prefix for webhook HMAC secrets."
}
