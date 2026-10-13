terraform {
  required_providers {
    lrcicd = {
      source = "registry.terraform.io/vaelatern/lrcicd"
    }
    nomad = {
      source = "hashicorp/nomad"
    }
    random = {
      source = "hashicorp/random"
    }
    gitlab = {
      source = "gitlabhq/gitlab"
    }
  }
}

resource "random_password" "auth_token" {
  length  = 48
  special = false
}

resource "random_password" "webhook_secret" {
  length  = 40
  special = false
}

locals {
  token = "vaelci-${random_password.auth_token.result}"
  keys_yaml = <<-EOT
${local.token}:
  - 'KICKOFF /${var.name}/.*'
  - 'POST /sync/${var.name}/.*'
  - 'GET /sync/${var.name}'
  - 'DELETE /sync/${var.name}'
  - 'POST /hooks/gitlab/${var.name}'
EOT
  shared_yaml = <<-EOT
- provider: gitlab
  key: "${random_password.webhook_secret.result}"
  valid-repo: "${var.name}"
EOT
}

resource "nomad_variable" "keys_d" {
  path = "${var.nomad_keys_d_prefix}/${var.name}"
  items = {
    "yaml" = local.keys_yaml
    "token" = local.token
  }
}

resource "nomad_variable" "shared_secrets" {
  path = "${var.nomad_shared_secrets_prefix}/${var.name}"
  items = {
    "yaml" = local.shared_yaml
  }
}

resource "lrcicd_repo" "this" {
  name                    = var.name
  url                     = var.git_url
  ssh_reading_private_key = var.ssh_reading_private_key
}

resource "gitlab_project_hook" "kickoff" {
  project                   = var.gitlab_project
  url                       = "${var.kickoff_url}/hooks/gitlab/${var.name}?token=${urlencode(local.token)}"
  token                     = random_password.webhook_secret.result
  push_events               = true
  enable_ssl_verification   = true
}
