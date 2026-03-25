job "temporal-cicd" {
  datacenters = ["dc1"]
  type        = "service"

  group "kickoff" {
    count = 1

    network {
      mode = "bridge"
      port "http" {}
    }

    service {
      name = "kickoff"
      port = "http"
    }

    task "kickoff" {
      driver = "docker"

      config {
        image        = "temporal-cicd-kickoff:2026-02-25"
        ports        = ["http"]
	volumes      = ["custom-kickoff:/custom-kickoff",
	                "keys.d:/keys.d"]
      }

      template {
        change_mode = "signal"
        change_signal = "SIGUSR1"
        data        = <<EOF
{{ range nomadVarList "temporal-cicd/keys-d" }}
# {{ .Path }}
{{ with nomadVar .Path }}
{{ .yaml.Value }}
{{ end }}
{{ end }}
EOF
        destination = "keys.d/nomad-secret-keys"
      }

      env {
        TEMPORAL_ADDRESS="172.17.0.1:7233"
      }
    }
  }

  group "cache" {
    count = 1

    network {
      mode = "bridge"
      port "http" {
        static = 8081
      }
    }

    service {
      name = "cache"
      port = "http"
    }

    task "cache" {
      driver = "docker"

      config {
        image   = "temporal-cicd-cache:2026-02-25"
        ports   = ["http"]
        volumes = ["repos:/repos", "ssh-keys:/ssh-keys", "keys.d:/keys.d"]
      }

      template {
        change_mode = "signal"
        change_signal = "SIGUSR1"
        data        = <<EOF
{{ range nomadVarList "temporal-cicd/keys-d" }}
# {{ .Path }}
{{ with nomadVar .Path }}
{{ .yaml.Value }}
{{ end }}
{{ end }}
EOF
        destination = "keys.d/nomad-secret-keys"
      }

      env {
        TEMPORAL_ADDRESS="172.17.0.1:7233"
      }
    }
  }

  group "artifacts" {
    count = 1

    network {
      mode = "bridge"
      port "http" {}
    }

    service {
      name = "artifacts"
      port = "http"
    }

    task "artifacts" {
      driver = "docker"

      config {
        image   = "temporal-cicd-artifacts:2026-02-25"
        ports   = ["http"]
        volumes = ["artifacts:/artifacts", "keys.d:/keys.d"]
      }

      template {
        change_mode = "signal"
        change_signal = "SIGUSR1"
        data        = <<EOF
{{ range nomadVarList "temporal-cicd/keys-d" }}
# {{ .Path }}
{{ with nomadVar .Path }}
{{ .yaml.Value }}
{{ end }}
{{ end }}
EOF
        destination = "keys.d/nomad-secret-keys"
      }

      env {
        TEMPORAL_ADDRESS="172.17.0.1:7233"
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
        image   = "temporal-cicd-builder:2026-02-25"
      }

      env {
        TEMPORAL_ADDRESS="172.17.0.1:7233"
      }
    }
  }
}
