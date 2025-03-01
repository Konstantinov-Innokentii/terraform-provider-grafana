---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "grafana_dashboards Data Source - terraform-provider-grafana"
subcategory: ""
description: |-
  Datasource for retrieving all dashboards. Specify list of folder IDs to search in for dashboards.
  Official documentation https://grafana.com/docs/grafana/latest/dashboards/Folder/Dashboard Search HTTP API https://grafana.com/docs/grafana/latest/http_api/folder_dashboard_search/Dashboard HTTP API https://grafana.com/docs/grafana/latest/http_api/dashboard/
---

# grafana_dashboards (Data Source)

Datasource for retrieving all dashboards. Specify list of folder IDs to search in for dashboards.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
* [Folder/Dashboard Search HTTP API](https://grafana.com/docs/grafana/latest/http_api/folder_dashboard_search/)
* [Dashboard HTTP API](https://grafana.com/docs/grafana/latest/http_api/dashboard/)

## Example Usage

```terraform
resource "grafana_folder" "data_source_dashboards" {
  title = "test folder data_source_dashboards"
}

// retrieve dashboards by tags, folderIDs, or both
resource "grafana_dashboard" "data_source_dashboards1" {
  folder = grafana_folder.data_source_dashboards.id
  config_json = jsonencode({
    id            = 23456
    title         = "data_source_dashboards 1"
    tags          = ["data_source_dashboards"]
    timezone      = "browser"
    schemaVersion = 16
  })
}

data "grafana_dashboards" "tags" {
  tags = jsondecode(grafana_dashboard.data_source_dashboards1.config_json)["tags"]
}

data "grafana_dashboards" "folder_ids" {
  folder_ids = [grafana_dashboard.data_source_dashboards1.folder]
}

data "grafana_dashboards" "folder_ids_tags" {
  folder_ids = [grafana_dashboard.data_source_dashboards1.folder]
  tags       = jsondecode(grafana_dashboard.data_source_dashboards1.config_json)["tags"]
}

resource "grafana_dashboard" "data_source_dashboards2" {
  folder = 0 // General folder
  config_json = jsonencode({
    id            = 23456
    title         = "data_source_dashboards 2"
    tags          = ["prod"]
    timezone      = "browser"
    schemaVersion = 16
  })
}

// use depends_on to wait for dashboard resource to be created before searching
data "grafana_dashboards" "all" {
  depends_on = [
    grafana_dashboard.data_source_dashboards1,
    grafana_dashboard.data_source_dashboards2
  ]
}

data "grafana_dashboard" "from_data_source" {
  uid = data.grafana_dashboards.all.dashboards[0].uid
}

// get only one result
data "grafana_dashboards" "limit_one" {
  limit = 1
  depends_on = [
    grafana_dashboard.data_source_dashboards1,
    grafana_dashboard.data_source_dashboards2
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- **folder_ids** (List of Number) Numerical IDs of Grafana folders containing dashboards. Specify to filter for dashboards by folder (eg. `[0]` for General folder), or leave blank to get all dashboards in all folders.
- **id** (String) The ID of this resource.
- **limit** (Number) Maximum number of dashboard search results to return. Defaults to `5000`.
- **tags** (List of String) List of string Grafana dashboard tags to search for, eg. `["prod"]`. Used only as search input, i.e., attribute value will remain unchanged.

### Read-Only

- **dashboards** (List of Object) (see [below for nested schema](#nestedatt--dashboards))

<a id="nestedatt--dashboards"></a>
### Nested Schema for `dashboards`

Read-Only:

- **folder_title** (String)
- **title** (String)
- **uid** (String)


