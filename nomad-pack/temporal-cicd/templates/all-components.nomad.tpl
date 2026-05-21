#!/usr/bin/env -S nomad job run

[[ define "auth-keys-d" ]]
      template {
        change_mode   = "signal"
        change_signal = "SIGUSR1"
        data          = <<EOF
{{ range nomadVarList [[ dig "keys-d-path" "temporal-cicd/keys-d" .Args | quote ]] }}
# {{ .Path }}
{{ with nomadVar .Path }}
{{ .yaml.Value }}
{{ end }}
{{ end }}
EOF
        destination   = "keys.d/nomad-secret-keys.yml"
      }
[[ end ]]

[[ define "temporal-address-env-file" ]]
      template {
	env  = true
        data = <<EOF
TEMPORAL_ADDRESS="[[ dig "hookups" "temporal" "{{ range nomadService 1 (env \"NOMAD_ALLOC_ID\") \"temporal-frontend\" }}{{ .Address }}:{{ .Port }}{{ end }}" .Args ]]"
EOF
        destination   = "local/env/temporal-frontend"
      }
[[ end ]]

[[ define "config-file" ]]
      template {
	env  = true
        data = <<EOF
TCD_CONFIG_FILE=${NOMAD_TASK_DIR}/config.json
EOF
        destination   = "local/env/configuration"
      }

      template {
        data = <<EOF
[[ dig "config" (dict) .Args | tojson ]]
EOF
        destination   = "local/config.json"
      }
[[ end ]]

job [[ getarg "jobname" .Args ]] {
  type        = "service"
  datacenters = [[ getarg "datacenters" .Args ]]

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
        image = "ghcr.io/vaelatern/temporal-cicd/kickoff:[[ dig "version" "kickoff" (dig "version" "default" "master" .Args) .Args ]]"
        ports = ["http"]
        volumes = ["custom-kickoff:/custom-kickoff",
        "keys.d:/keys"]
      }

      [[ template "config-file" . ]]

      [[ template "auth-keys-d" . ]]

      [[ template "temporal-address-env-file" . ]]
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
        image   = "ghcr.io/vaelatern/temporal-cicd/cache:[[ dig "version" "cache" (dig "version" "default" "master" .Args) .Args ]]"
        ports   = ["http"]
        volumes = ["repos:/repos", "ssh-keys:/ssh-keys", "keys.d:/keys"]
      }

      [[ template "config-file" . ]]

      [[ template "auth-keys-d" . ]]

      [[ template "temporal-address-env-file" . ]]
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
        image   = "ghcr.io/vaelatern/temporal-cicd/artifacts:[[ dig "version" "artifacts" (dig "version" "default" "master" .Args) .Args ]]"
        ports   = ["http"]
        volumes = ["artifacts:/artifacts", "keys.d:/keys"]
      }

      [[ template "config-file" . ]]

      [[ template "auth-keys-d" . ]]

      [[ template "temporal-address-env-file" . ]]
    }
  }

  group "builder" {
    count = 1

    network {
      mode = "bridge"
    }

    task "builder" {
      driver = "oci-chroot"

      config {
        image = "ghcr.io/vaelatern/temporal-cicd/builder:[[ dig "version" "builder" (dig "version" "default" "master" .Args) .Args ]]"
        bind_sockets = ["/var/run/docker.sock"]
      }

      [[ template "config-file" . ]]

      [[ template "temporal-address-env-file" . ]]

      template {
	env = true
        data        = <<EOF
TCD_CACHE_URL="http://{{ range nomadService 1 (env "NOMAD_ALLOC_ID") "[[ getarg "jobname" .Args | unquote ]]-cache" }}{{ .Address }}:{{ .Port }}{{ end }}"
TCD_ARTIFACTS_URL="{{ range nomadService 1 (env "NOMAD_ALLOC_ID") "[[ getarg "jobname" .Args | unquote ]]-artifacts" }}{{ .Address }}:{{ .Port }}{{ end }}"
EOF
        destination = "local/env/service-discovery"
      }
    }
  }
}
