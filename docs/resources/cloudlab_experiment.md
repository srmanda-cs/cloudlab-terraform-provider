---
page_title: "cloudlab_experiment Resource - terraform-provider-cloudlab"
description: |-
  Manages a CloudLab experiment. Creating an experiment provisions physical or virtual machines on the CloudLab testbed.
---

# cloudlab_experiment (Resource)

Manages a CloudLab experiment. In CloudLab terminology, an "experiment" is a running instantiation of a profile — it represents actual allocated hardware (or VMs) with assigned IP addresses and hostnames.

Creating this resource calls `POST /experiments` and (by default) waits up to 30 minutes for the experiment to reach `ready` status. Deleting the resource terminates the experiment and releases all associated hardware.

**Mutable without replacement:** `expires_at` (absolute expiry), `extend_by` (relative hours), `extend_reason`, and `bindings`. All other configuration attributes require destroying and recreating the experiment.

## Example Usage

### Basic experiment using a public profile

```terraform
resource "cloudlab_experiment" "cluster" {
  name            = "my-cluster"
  project         = "MyProject"
  profile_name    = "small-lan"
  profile_project = "CloudLab"
  duration        = 24
}

output "experiment_id" {
  value = cloudlab_experiment.cluster.id
}

output "experiment_status" {
  value = cloudlab_experiment.cluster.status
}
```

### Experiment from a custom profile with parameter bindings

```terraform
resource "cloudlab_profile" "topology" {
  name    = "my-topology"
  project = "MyProject"
  script  = file("${path.module}/profile.py")
}

resource "cloudlab_experiment" "run" {
  name            = "experiment-run1"
  project         = "MyProject"
  profile_name    = cloudlab_profile.topology.name
  profile_project = cloudlab_profile.topology.project
  duration        = 48

  # Pass parameter bindings to parameterized profiles
  bindings = {
    n_nodes   = "4"
    node_type = "xl170"
  }
}
```

### Scheduled experiment with SSH key injection

```terraform
resource "cloudlab_experiment" "scheduled" {
  name            = "nightly-run"
  project         = "MyProject"
  profile_name    = "test-profile"
  profile_project = "MyProject"
  start_at        = "2026-03-01T02:00:00Z"
  stop_at         = "2026-03-01T08:00:00Z"
  sshpubkey       = file("~/.ssh/id_ed25519.pub")
  wait_for_ready  = false
}
```

### Extending an experiment's lifetime (absolute timestamp)

```terraform
resource "cloudlab_experiment" "cluster" {
  name            = "my-cluster"
  project         = "MyProject"
  profile_name    = "small-lan"
  profile_project = "CloudLab"
  duration        = 24
  expires_at      = "2026-04-01T00:00:00Z"
  extend_reason   = "Need more time for data collection"
}
```

### Extending an experiment's lifetime (relative hours)

```terraform
resource "cloudlab_experiment" "cluster" {
  name            = "my-cluster"
  project         = "MyProject"
  profile_name    = "small-lan"
  profile_project = "CloudLab"
  duration        = 24
  # Adds 24 h to the current expiration. Change to 48 in a later apply to add another 24 h.
  extend_by     = 24
  extend_reason = "Need more time for benchmarking"
}
```

### Using a repository-backed profile with a specific refspec

```terraform
resource "cloudlab_experiment" "from_branch" {
  name            = "feature-test"
  project         = "MyProject"
  profile_name    = "my-repo-profile"
  profile_project = "MyProject"
  refspec         = "refs/heads/feature-branch"
  duration        = 12
}
```

### Retrieve node details after creation

```terraform
resource "cloudlab_experiment" "cluster" {
  name            = "my-cluster"
  project         = "MyProject"
  profile_name    = "small-lan"
  profile_project = "CloudLab"
}

# Get all node hostnames via the manifest data source
data "cloudlab_manifest" "nodes" {
  experiment_id = cloudlab_experiment.cluster.id
}

output "hostnames" {
  value = flatten([
    for m in data.cloudlab_manifest.nodes.manifests :
    [for n in m.nodes : n.hostname]
  ])
}

# Or look up a specific node
data "cloudlab_node" "node1" {
  experiment_id = cloudlab_experiment.cluster.id
  client_id     = "node1"
}

output "node1_ipv4" {
  value = data.cloudlab_node.node1.ipv4
}
```

## Schema

### Required

- `name` (String) — A human-readable name for the experiment. Must be unique within the project. **Forces new resource.**
- `project` (String) — The CloudLab project to instantiate the experiment in. **Forces new resource.**
- `profile_name` (String) — The name of the profile (topology template) used to create the experiment. **Forces new resource.**
- `profile_project` (String) — The project that owns the profile. **Forces new resource.**

### Optional

- `group` (String) — The project subgroup to instantiate the experiment in. **Forces new resource.**
- `duration` (Number) — Initial experiment duration in hours. **Forces new resource.**
- `start_at` (String) — Schedule the experiment to start at a future time (RFC3339 format, e.g. `2026-03-01T02:00:00Z`). Validated at plan time. **Forces new resource.**
- `stop_at` (String) — Schedule the experiment to stop at a future time (RFC3339 format). Validated at plan time. **Forces new resource.**
- `paramset_name` (String) — Name of a saved parameter set to apply to the profile. **Forces new resource.**
- `paramset_owner` (String) — The owner (username) of the parameter set. **Forces new resource.**
- `bindings` (Map of String) — Parameter bindings to apply to the profile as a native HCL map. Values must be strings; use `tostring()` for numeric parameters (e.g. `{ n_nodes = "4", node_type = "xl170" }`). Mutable: changing this value performs a `PATCH /experiments/{id}` to apply new bindings to the running experiment.
- `refspec` (String) — For repository-backed profiles, optionally specify a `refspec[:hash]` to use instead of the HEAD of the default branch. **Forces new resource.**
- `sshpubkey` (String) — An additional SSH public key to install on all nodes in the experiment. **Forces new resource.**
- `expires_at` (String) — The time the experiment should expire (RFC3339 format). Validated at plan time. Setting or changing this value performs a `PUT /experiments/{id}` to set an absolute expiry. Mutually exclusive in intent with `extend_by` — use one or the other per apply.
- `extend_by` (Number) — Number of hours to add to the experiment's current expiration. Changing this value performs a `PUT /experiments/{id}` with `extend_by`. To extend by additional hours in a later apply, increment the value (e.g. `24` → `48` means "48 h total added in this config"). Mutually exclusive in intent with `expires_at`.
- `extend_reason` (String) — Optional reason text to include when extending the experiment's lifetime via `expires_at` or `extend_by`.
- `wait_for_ready` (Boolean) — If `true` (default), Terraform blocks until the experiment reaches `ready` status before completing the `apply`. Set to `false` to return immediately after the create request is submitted. The provider polls every 15 seconds with a 30-minute timeout.

### Read-Only

- `id` (String) — The unique identifier (UUID) of the experiment assigned by CloudLab.
- `creator` (String) — The CloudLab username who created the experiment.
- `updater` (String) — The CloudLab username who last updated the experiment (nullable).
- `status` (String) — The current status of the experiment (e.g., `created`, `ready`, `failed`).
- `created_at` (String) — The timestamp when the experiment was created (ISO 8601).
- `started_at` (String) — The timestamp when the experiment was actually started (ISO 8601, nullable).
- `expires_at` (String) — The current expiration time of the experiment (ISO 8601, nullable). Also writable — see above.
- `url` (String) — The URL of the Portal status page for this experiment.
- `wbstore_id` (String) — The ID of the experiment's WB store (internal CloudLab identifier).
- `repository_url` (String) — The repository URL (for repository-backed profiles, nullable).
- `repository_refspec` (String) — The refspec used for the experiment (for repository-backed profiles, nullable).
- `repository_hash` (String) — The commit hash used for the experiment (for repository-backed profiles, nullable).

## Timeouts

When `wait_for_ready = true` (the default), the provider polls every **15 seconds** with a maximum timeout of **30 minutes**. If the experiment reaches `failed` status during polling, Terraform returns an error immediately.

## Import

Experiments can be imported using their UUID:

```shell
terraform import cloudlab_experiment.cluster <experiment-uuid>
```
