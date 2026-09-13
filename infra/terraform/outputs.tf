output "elasticsearch_url" {
  value = "http://localhost:${var.es_external_port}"
}

output "neo4j_browser_url" {
  value = "http://localhost:${var.neo4j_http_external_port}"
}

output "neo4j_bolt_url" {
  value = "bolt://localhost:${var.neo4j_bolt_external_port}"
}

output "wraith_backend_url" {
  value = "http://localhost:${var.backend_external_port}"
}
