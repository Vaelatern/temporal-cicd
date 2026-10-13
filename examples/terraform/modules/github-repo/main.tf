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
    github = {
      source = "integrations/github"
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
  - 'POST /hooks/github/${var.name}'
EOT
  shared_yaml = <<-EOT
- provider: github
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
  name                     = var.name
  url                      = var.git_url
  ssh_reading_private_key  = var.ssh_reading_private_key
}

resource "github_repository_webhook" "kickoff" {
  repository = var.github_repository
  active     = true
  events     = ["push"]

  configuration {
    url          = "${var.kickoff_url}/hooks/github/${var.name}?token=${urlencode(local.token)}"
    content_type = "json"
    insecure_ssl = false
    secret       = random_password.webhook_secret.result
  }
}
