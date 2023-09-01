
# data "kubernetes_namespace" "istio_system" {
#   metadata {
#     name = "istio-system"
#   }
# }

resource "helm_release" "istio" {
  name             = var.deployment_name
  repository       = var.repository
  chart            = "base"
  namespace        = var.namespace
  version          = "1.18.2"
  create_namespace = true

  set {
    name  = "defaultRevision"
    value = "default"
  }

  # depends_on = [data.kubernetes_namespace.istio_system]
}

resource "helm_release" "istiod" {
  name       = "isitod-release"
  repository = var.repository
  chart      = "istiod"
  namespace  = var.namespace
  version    = "1.18.2"

  depends_on = [helm_release.istio]
}

resource "helm_release" "istio_gateway" {
  name       = "istio-gateway"
  repository = var.repository
  chart      = "gateway"
  namespace  = var.namespace
  version     = "1.18.2"
  depends_on = [helm_release.istio]
}

