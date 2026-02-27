data "cloudlab_node" "node1" {
  experiment_id = cloudlab_experiment.my_exp.id
  client_id     = "node1"
}

output "node_hostname" {
  value = data.cloudlab_node.node1.hostname
}

output "node_ipv4" {
  value = data.cloudlab_node.node1.ipv4
}
