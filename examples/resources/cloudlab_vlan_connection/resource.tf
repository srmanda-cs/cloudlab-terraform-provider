resource "cloudlab_vlan_connection" "link" {
  experiment_id = cloudlab_experiment.source.id
  source_lan    = "shared-lan"
  target_id     = cloudlab_experiment.target.id
  target_lan    = "shared-lan"
}
