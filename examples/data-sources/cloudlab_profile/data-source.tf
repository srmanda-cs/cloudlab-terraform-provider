data "cloudlab_profile" "existing" {
  id = "myproject,myprofile"
}

resource "cloudlab_experiment" "exp" {
  name            = "my-experiment"
  project         = "myproject"
  profile_name    = data.cloudlab_profile.existing.name
  profile_project = data.cloudlab_profile.existing.project
}
