terraform {
  required_version = ">= 1.5.0"
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.0"
    }
  }
}

provider "docker" {}

# --- Networking ---
resource "docker_network" "wraith" {
  name = "wraith-net"
}

# --- Elasticsearch: the production/staging SIEM target rules deploy to
#     after passing CI. (Ephemeral per-PR test instances are provisioned
#     directly via the Go orchestrator/Docker SDK, not Terraform, since
#     they need sub-second create/destroy cycles inside CI.) ---
resource "docker_image" "elasticsearch" {
  name = "docker.elastic.co/elasticsearch/elasticsearch:8.13.0"
}

resource "docker_volume" "es_data" {
  name = "wraith-es-data-${var.environment}"
}

resource "docker_container" "elasticsearch" {
  name  = "wraith-es-${var.environment}"
  image = docker_image.elasticsearch.image_id

  networks_advanced {
    name = docker_network.wraith.name
  }

  env = [
    "discovery.type=single-node",
    "xpack.security.enabled=${var.es_security_enabled}",
    "ES_JAVA_OPTS=-Xms${var.es_heap_size} -Xmx${var.es_heap_size}",
  ]

  ports {
    internal = 9200
    external = var.es_external_port
  }

  volumes {
    volume_name    = docker_volume.es_data.name
    container_path = "/usr/share/elasticsearch/data"
  }

  restart = "unless-stopped"

  healthcheck {
    test     = ["CMD-SHELL", "curl -sf http://localhost:9200/_cluster/health || exit 1"]
    interval = "30s"
    timeout  = "5s"
    retries  = 5
  }
}

# --- Neo4j: persistent graph store for MITRE ATT&CK attack-path mapping
#     used in production dashboards (as opposed to the per-run ephemeral
#     graphs built during CI testing). ---
resource "docker_image" "neo4j" {
  name = "neo4j:5.19-community"
}

resource "docker_volume" "neo4j_data" {
  name = "wraith-neo4j-data-${var.environment}"
}

resource "docker_container" "neo4j" {
  name  = "wraith-neo4j-${var.environment}"
  image = docker_image.neo4j.image_id

  networks_advanced {
    name = docker_network.wraith.name
  }

  env = [
    "NEO4J_AUTH=neo4j/${var.neo4j_password}",
    "NEO4J_PLUGINS=[\"apoc\"]",
  ]

  ports {
    internal = 7474
    external = var.neo4j_http_external_port
  }
  ports {
    internal = 7687
    external = var.neo4j_bolt_external_port
  }

  volumes {
    volume_name    = docker_volume.neo4j_data.name
    container_path = "/data"
  }

  restart = "unless-stopped"
}

# --- WRAITH Go API server ---
resource "docker_image" "wraith_backend" {
  name = var.wraith_backend_image
}

resource "docker_container" "wraith_backend" {
  name  = "wraith-backend-${var.environment}"
  image = docker_image.wraith_backend.image_id

  networks_advanced {
    name = docker_network.wraith.name
  }

  env = [
    "WRAITH_WEBHOOK_SECRET=${var.webhook_secret}",
    "WRAITH_ES_ADDR=http://wraith-es-${var.environment}:9200",
    "WRAITH_NEO4J_ADDR=bolt://wraith-neo4j-${var.environment}:7687",
    "WRAITH_PORT=8080",
  ]

  ports {
    internal = 8080
    external = var.backend_external_port
  }

  restart   = "unless-stopped"
  depends_on = [docker_container.elasticsearch, docker_container.neo4j]
}
