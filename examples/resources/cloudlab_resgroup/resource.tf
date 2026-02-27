resource "cloudlab_resgroup" "hw_reservation" {
  project    = "MyProject"
  reason     = "Weekly experiment run requiring guaranteed xl170 hardware"
  expires_at = "2026-03-01T00:00:00Z"

  node_types = [
    {
      urn       = "urn:publicid:IDN+utah.cloudlab.us+authority+cm"
      node_type = "xl170"
      count     = 4
    }
  ]
}
