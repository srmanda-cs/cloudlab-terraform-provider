# cloudlab_resgroup

Queries an existing CloudLab reservation group by its UUID. Use this data source to reference reservation groups that were created outside of Terraform or in a separate Terraform state.

## Example Usage

```hcl
data "cloudlab_resgroup" "existing" {
  id = "a194e2be-1e5b-4617-84de-c4966cb5c578"
}

output "resgroup_expires_at" {
  value = data.cloudlab_resgroup.existing.expires_at
}
```

## Argument Reference

* `id` - (Required) The unique identifier (UUID) of the reservation group to look up.

## Attribute Reference

* `project` - The CloudLab project this reservation group belongs to.
* `group` - The project subgroup this reservation group belongs to.
* `reason` - The reason the reservation was created.
* `creator` - The CloudLab username who created the reservation group.
* `created_at` - The timestamp when the reservation group was created.
* `start_at` - The time the reservation starts.
* `expires_at` - The time the reservation expires.
* `powder_zones` - The Powder zone for radio reservations.
* `node_types` - The list of node type reservations. Each entry contains:
  * `urn` - The aggregate URN of the reservation.
  * `node_type` - The hardware node type reserved.
  * `count` - The number of nodes reserved.
* `ranges` - The list of frequency range reservations. Each entry contains:
  * `min_freq` - The start of the frequency range (inclusive) in MHz.
  * `max_freq` - The end of the frequency range (inclusive) in MHz.
* `routes` - The list of named route reservations. Each entry contains:
  * `name` - The route name reserved.
