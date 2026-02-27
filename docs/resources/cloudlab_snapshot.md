# cloudlab_snapshot

Manages a CloudLab node image snapshot. Creates an image snapshot of a running node in an experiment. The image can then be used as a base image in future experiments.

> **Note:** Destroying this resource removes it from Terraform state only. The created image persists in CloudLab and must be deleted manually through the portal.

## Example Usage

```hcl
resource "cloudlab_snapshot" "my_image" {
  experiment_id    = cloudlab_experiment.my_exp.id
  client_id        = "node1"
  image_name       = "my-custom-image"
  whole_disk       = false
  wait_for_complete = true
}

output "image_urn" {
  value = cloudlab_snapshot.my_image.image_urn
}
```

## Argument Reference

* `experiment_id` - (Required, Forces New) The UUID of the running experiment containing the node to snapshot.
* `client_id` - (Required, Forces New) The logical name (client ID) of the node to snapshot.
* `image_name` - (Required, Forces New) The name of the image to create or update.
* `whole_disk` - (Optional, Forces New) If true, take a whole disk image. Defaults to `false` (partition image).
* `wait_for_complete` - (Optional) If true (default), Terraform will wait until the snapshot completes before finishing. Set to `false` to return immediately after the snapshot is initiated.

## Attribute Reference

* `id` - The unique identifier (UUID) of the snapshot request.
* `status` - The current status of the snapshot operation.
* `status_timestamp` - The timestamp of the last status update.
* `image_size` - The current size of the image in KB.
* `image_urn` - The URN of the created image.
* `error_message` - Error message if the snapshot failed.
