#!/usr/bin/env -S nomad job run
job "temporal-cicd" {
  datacenters = ["dc1"]
  type        = "service"

  group "kickoff" {
    count = 1

    network {
      mode = "bridge"
      port "http" {
	to = 8080
      }
    }

    service {
      provider="nomad"
      port = "http"
    }

    task "kickoff" {
      driver = "docker"

      config {
        image = "ghcr.io/vaelatern/temporal-cicd/kickoff:master"
        ports = ["http"]
        volumes = ["custom-kickoff:/custom-kickoff",
        "keys.d:/keys"]
      }

      template {
        change_mode   = "signal"
        change_signal = "SIGUSR1"
        data          = <<EOF
{{ range nomadVarList "temporal-cicd/keys-d" }}
# {{ .Path }}
{{ with nomadVar .Path }}
{{ .yaml.Value }}
{{ end }}
{{ end }}
EOF
        destination   = "keys.d/nomad-secret-keys.yml"
      }

      env {
        TEMPORAL_ADDRESS = "172.17.0.1:7233"
      }
    }
  }

  group "cache" {
    count = 1

    network {
      mode = "bridge"
      port "http" {
	to = 8080
      }
    }

    service {
      provider="nomad"
      port = "http"
    }

    task "cache" {
      driver = "docker"

      config {
        image   = "ghcr.io/vaelatern/temporal-cicd/cache:master"
        ports   = ["http"]
        volumes = ["repos:/repos", "ssh-keys:/ssh-keys", "keys.d:/keys"]
      }

      template {
        change_mode   = "signal"
        change_signal = "SIGUSR1"
        data          = <<EOF
{{ range nomadVarList "temporal-cicd/keys-d" }}
# {{ .Path }}
{{ with nomadVar .Path }}
{{ .yaml.Value }}
{{ end }}
{{ end }}
EOF
        destination   = "keys.d/nomad-secret-keys.yml"
      }

      env {
        TEMPORAL_ADDRESS = "172.17.0.1:7233"
      }
    }
  }

  group "artifacts" {
    count = 1

    network {
      mode = "bridge"
      port "http" {
	to = 8080
      }
    }

    service {
      provider="nomad"
      port = "http"
    }

    task "artifacts" {
      driver = "docker"

      config {
        image   = "ghcr.io/vaelatern/temporal-cicd/artifacts:master"
        ports   = ["http"]
        volumes = ["artifacts:/artifacts", "keys.d:/keys"]
      }

      template {
        change_mode   = "signal"
        change_signal = "SIGUSR1"
        data          = <<EOF
{{ range nomadVarList "temporal-cicd/keys-d" }}
# {{ .Path }}
{{ with nomadVar .Path }}
{{ .yaml.Value }}
{{ end }}
{{ end }}
EOF
        destination   = "keys.d/nomad-secret-keys.yml"
      }

      env {
        TEMPORAL_ADDRESS = "172.17.0.1:7233"
      }
    }
  }

  group "builder" {
    count = 1

    network {
      mode = "bridge"
    }

    task "builder" {
      driver = "docker"

      config {
        image = "ghcr.io/vaelatern/temporal-cicd/builder:master"
      }

      env {
        TEMPORAL_ADDRESS = "172.17.0.1:7233"
      }

      template {
	env = true
        data        = <<EOF
CACHE_ADDR="{{ range nomadService 1 (env "NOMAD_ALLOC_ID") "temporal-cicd-cache" }}{{ .Address }}:{{ .Port }}{{ end }}"
EOF
        destination = "service-discovery.env"
      }
    }
  }
}
