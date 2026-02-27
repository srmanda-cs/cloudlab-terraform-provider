resource "cloudlab_snapshot" "my_image" {
  experiment_id     = cloudlab_experiment.my_exp.id
  client_id         = "node1"
  image_name        = "my-custom-image"
  whole_disk        = false
  wait_for_complete = true
}

output "image_urn" {
  value = cloudlab_snapshot.my_image.image_urn
}
