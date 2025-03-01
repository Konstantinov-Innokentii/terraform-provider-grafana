---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "grafana_synthetic_monitoring_installation Resource - terraform-provider-grafana"
subcategory: ""
description: |-
  Sets up Synthetic Monitoring on a Grafana cloud stack and generates a token.
  Once a Grafana Cloud stack is created, a user can either use this resource or go into the UI to install synthetic monitoring.
  This resource cannot be imported but it can be used on an existing Synthetic Monitoring installation without issues.
  Official documentation https://grafana.com/docs/grafana-cloud/synthetic-monitoring/installation/API documentation https://github.com/grafana/synthetic-monitoring-api-go-client/blob/main/docs/API.md#apiv1registerinstall
---

# grafana_synthetic_monitoring_installation (Resource)

Sets up Synthetic Monitoring on a Grafana cloud stack and generates a token. 
Once a Grafana Cloud stack is created, a user can either use this resource or go into the UI to install synthetic monitoring.
This resource cannot be imported but it can be used on an existing Synthetic Monitoring installation without issues.

* [Official documentation](https://grafana.com/docs/grafana-cloud/synthetic-monitoring/installation/)
* [API documentation](https://github.com/grafana/synthetic-monitoring-api-go-client/blob/main/docs/API.md#apiv1registerinstall)

## Example Usage

```terraform
resource "grafana_cloud_stack" "sm_stack" {
  name        = "<stack-name>"
  slug        = "<stack-slug>"
  region_slug = "us"
}

resource "grafana_cloud_api_key" "metrics_publish" {
  name           = "MetricsPublisherForSM"
  role           = "MetricsPublisher"
  cloud_org_slug = "<org-slug>"
}

resource "grafana_synthetic_monitoring_installation" "sm_stack" {
  stack_id              = grafana_cloud_stack.sm_stack.id
  metrics_instance_id   = grafana_cloud_stack.sm_stack.prometheus_user_id
  logs_instance_id      = grafana_cloud_stack.sm_stack.logs_user_id
  metrics_publisher_key = grafana_cloud_api_key.metrics_publish.key
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- **logs_instance_id** (Number) The ID of the logs instance to install SM on (stack's `logs_user_id` attribute).
- **metrics_instance_id** (Number) The ID of the metrics instance to install SM on (stack's `prometheus_user_id` attribute).
- **metrics_publisher_key** (String, Sensitive) The Cloud API Key with the `MetricsPublisher` role used to publish metrics to the SM API
- **stack_id** (Number) The ID of the stack to install SM on.

### Optional

- **id** (String) The ID of this resource.

### Read-Only

- **sm_access_token** (String) Generated token to access the SM API.


