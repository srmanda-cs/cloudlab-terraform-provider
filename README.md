# Terraform Provider for CloudLab

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

The **CloudLab Terraform Provider** allows [Terraform](https://terraform.io) to manage resources on [CloudLab](https://www.cloudlab.us/) — the academic cloud and network testbed operated by the University of Utah, Clemson University, and the University of Wisconsin.

> **Status:** Under active development. Not yet published to the HashiCorp Registry.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21 (to build from source)
- A CloudLab account with a valid API token

## Getting a CloudLab API Token

1. Log in to [cloudlab.us](https://www.cloudlab.us/)
2. Navigate to your profile settings
3. Generate an API token under **API Tokens**

## Usage

```hcl
terraform {
  required_providers {
    cloudlab = {
      source  = "srmanda-cs/cloudlab"
      version = "~> 0.1"
    }
  }
}

provider "cloudlab" {
  token = var.cloudlab_token
}

# Create a profile (topology template)
resource "cloudlab_profile" "small_cluster" {
  name    = "small-cluster"
  project = "MyProject"
  script  = file("profile.py")
}

# Spin up an experiment (provisions actual machines)
resource "cloudlab_experiment" "cluster" {
  name            = "my-cluster"
  project         = "MyProject"
  profile_name    = cloudlab_profile.small_cluster.name
  profile_project = "MyProject"
  duration        = 24
}

# Access node hostnames/IPs
output "nodes" {
  value = cloudlab_experiment.cluster.nodes
}
```

## Resources

| Resource | Description |
|----------|-------------|
| `cloudlab_experiment` | Provisions a set of machines on CloudLab |
| `cloudlab_profile` | Manages experiment profiles (topology templates) |
| `cloudlab_resgroup` | Manages hardware reservation groups |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `cloudlab_experiment` | Queries a running experiment by name or ID |
| `cloudlab_manifest` | Retrieves node hostnames and IPs from a running experiment |

## Development

```bash
git clone https://github.com/srmanda-cs/terraform-provider-cloudlab.git
cd terraform-provider-cloudlab
go build ./...
go test ./...
```

## License

[MIT License](LICENSE)
