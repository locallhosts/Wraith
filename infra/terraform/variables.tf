variable "environment" {
  description = "Deployment environment name (e.g. staging, prod)"
  type        = string
  default     = "staging"
}

variable "es_external_port" {
  type    = number
  default = 9200
}

variable "es_heap_size" {
  type    = string
  default = "1g"
}

variable "es_security_enabled" {
  description = "Set true in prod and pair with an ELASTIC_PASSWORD/TLS setup"
  type        = bool
  default     = false
}

variable "neo4j_password" {
  type      = string
  sensitive = true
}

variable "neo4j_http_external_port" {
  type    = number
  default = 7474
}

variable "neo4j_bolt_external_port" {
  type    = number
  default = 7687
}

variable "wraith_backend_image" {
  description = "Built image for backend-go, e.g. ghcr.io/yourname/wraith-backend:latest"
  type        = string
  default     = "ghcr.io/yourname/wraith-backend:latest"
}

variable "webhook_secret" {
  description = "GitHub webhook HMAC secret"
  type        = string
  sensitive   = true
}

variable "backend_external_port" {
  type    = number
  default = 8080
}
