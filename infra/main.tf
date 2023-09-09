terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "2.23.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "2.11.0"
    }
  }
}

locals {
  kubeconfig = "~/.kube/config"
}

provider "kubernetes" {
  config_path = local.kubeconfig
}

provider "helm" {
  kubernetes {
    config_path = local.kubeconfig
  }
}

module "istio" {
  source          = "./istio"
  deployment_name = "my-istio-release"
  repository      = "https://istio-release.storage.googleapis.com/charts"
}

module "nats" {
  source     = "./nats"
  repository = "./nats/nats-repo/helm/charts"
}
