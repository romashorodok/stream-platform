
variable "repository" {
  type    = string
  default = "./nats-repo/helm/charts"
}

variable "deployment_name" {
  type    = string
  default = "nats-release"
}

variable "namespace" {
  type    = string
  default = "nats-system"
}
