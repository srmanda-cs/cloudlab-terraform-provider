data "cloudlab_resgroup" "existing" {
  id = "a194e2be-1e5b-4617-84de-c4966cb5c578"
}

output "resgroup_expires_at" {
  value = data.cloudlab_resgroup.existing.expires_at
}
