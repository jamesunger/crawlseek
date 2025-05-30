
terraform {
  required_providers {
    scaleway = {
      source = "scaleway/scaleway"
    }
  }
  required_version = ">= 0.13"
}

provider "scaleway" {
  zone   = "fr-par-1"
  region = "fr-par"
}


resource "scaleway_k8s_cluster" "my_k8s_cluster" {
  name        = "crawlseek"

  # Define the Kubernetes version you would like
  version     = "1.30.5" # Check the Scaleway documentation for the latest versions

  # Define the Kapsule region
  region      = "fr-par"

  delete_additional_resources = false
  cni = "cilium"

  private_network_id          = scaleway_vpc_private_network.pn.id


}


resource "scaleway_vpc_private_network" "pn" {}


  # Define the node pools
resource "scaleway_k8s_pool"  "node_pool" {

    name = "crawlseek-1"
    cluster_id  = scaleway_k8s_cluster.my_k8s_cluster.id
    node_type = "DEV1-M"
    external_traffic_policy = "true"



    # Define the size of the nodes
    size = 3

    # You can specify the number of nodes in the pool
    count = 1

    # Optional settings
    tags = ["dev", "kubernetes"]


}

resource "scaleway_registry_namespace" "crawlseek" {
  name        = "crawlseek-cr"
  description = "Crawl seek container registry"
  is_public   = false
}

# Outputs
output "kubeconfig" {
  sensitive = true
  value = scaleway_k8s_cluster.my_k8s_cluster.kubeconfig
}

output "cluster_id" {
  sensitive = true
  value = scaleway_k8s_cluster.my_k8s_cluster.id
}

output "docker-registry" {
  value = scaleway_registry_namespace.crawlseek.endpoint
}

