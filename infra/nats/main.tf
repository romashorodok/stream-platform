
resource "helm_release" "nats" {
  name             = var.deployment_name
  repository       = var.repository
  chart            = "nats"
  namespace        = var.namespace
  create_namespace = true

  values = [yamlencode({
    config = {
      cluster = {
        enabled  = false,
        replicas = 1,
      },
      jetstream = {
        enabled = true,
      },
    },
    service = {
      merge = {
        spec = {
          type = "LoadBalancer",
        },
      },
    },
  })]
}
