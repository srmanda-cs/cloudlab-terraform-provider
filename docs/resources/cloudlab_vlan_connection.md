# cloudlab_vlan_connection

Manages a shared VLAN connection between two CloudLab experiments. Creates a layer-2 connection between a LAN in one experiment and a LAN in another experiment. Both experiments must be running and have shared VLANs configured in their profiles.

## Example Usage

```hcl
resource "cloudlab_vlan_connection" "link" {
  experiment_id = cloudlab_experiment.source.id
  source_lan    = "shared-lan"
  target_id     = cloudlab_experiment.target.id
  target_lan    = "shared-lan"
}
```

## Argument Reference

* `experiment_id` - (Required, Forces New) The UUID of the source experiment.
* `source_lan` - (Required, Forces New) The client ID of the LAN in the source experiment.
* `target_id` - (Required, Forces New) The UUID or `project,name` of the target experiment to connect to.
* `target_lan` - (Required, Forces New) The client ID of the LAN in the target experiment.

## Attribute Reference

* `id` - A synthetic identifier for this VLAN connection, formatted as `experiment_id/source_lan`.

## Import

VLAN connections cannot be imported because the CloudLab API does not expose a query endpoint for connection state.
