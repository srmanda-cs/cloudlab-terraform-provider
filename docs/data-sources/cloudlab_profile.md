# cloudlab_profile

Queries an existing CloudLab profile by its UUID or `project,name` identifier. Use this data source to reference profiles that were created outside of Terraform or in a separate Terraform state.

## Example Usage

```hcl
data "cloudlab_profile" "existing" {
  id = "myproject,myprofile"
}

resource "cloudlab_experiment" "exp" {
  name            = "my-experiment"
  project         = "myproject"
  profile_name    = data.cloudlab_profile.existing.name
  profile_project = data.cloudlab_profile.existing.project
}
```

## Argument Reference

* `id` - (Required) The unique identifier (UUID or `project,name`) of the profile to look up.

## Attribute Reference

* `name` - The name of the profile.
* `project` - The CloudLab project that owns the profile.
* `creator` - The CloudLab username who created the profile.
* `version` - The current version number of the profile.
* `created_at` - The timestamp when the profile was created.
* `updated_at` - The timestamp when the profile was last updated.
* `repository_url` - The URL of the repository (for repository-backed profiles).
* `repository_refspec` - The refspec of the profile (for repository-backed profiles).
* `repository_hash` - The commit hash of the profile (for repository-backed profiles).
* `repository_githook` - The Portal URL of the repository githook.
* `public` - Whether the profile can be instantiated by any CloudLab user.
* `project_writable` - Whether other members of the project can modify this profile.
