
variable "repository" {
  type    = string
  default = "https://istio-release.storage.googleapis.com/charts"
}

variable "deployment_name" {
  type    = string
  default = "my-default-istio-release"
}

variable "namespace" {
  type = string
  default = "istio-system"
}
