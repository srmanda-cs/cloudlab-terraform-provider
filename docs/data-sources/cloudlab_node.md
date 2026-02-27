# cloudlab_node

Queries a specific node in a running CloudLab experiment. Returns detailed node status including hostname, IP address, and operational state.

## Example Usage

```hcl
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
```

## Argument Reference

* `experiment_id` - (Required) The UUID of the running experiment.
* `client_id` - (Required) The logical name (client ID) of the node within the experiment.

## Attribute Reference

* `urn` - The URN of the node.
* `hostname` - The fully qualified hostname of the node.
* `ipv4` - The IPv4 address of the node.
* `status` - The current status of the node.
* `state` - The current state of the node.
* `rawstate` - The current raw state of the node.
* `startup_status` - The current status of the startup execution service.
