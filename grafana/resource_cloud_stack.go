package grafana

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var stackSlugRegex = regexp.MustCompile("^[a-z][a-z0-9]+$")

func ResourceCloudStack() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/reference/cloud-api/#stacks/)
`,

		CreateContext: CreateStack,
		UpdateContext: UpdateStack,
		DeleteContext: DeleteStack,
		ReadContext:   ReadStack,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The stack id assigned to this stack by Grafana.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of stack. Conventionally matches the url of the instance (e.g. “<stack_slug>.grafana.net”).",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of stack.",
			},
			"slug": {
				Type:     schema.TypeString,
				Required: true,
				Description: `
Subdomain that the Grafana instance will be available at (i.e. setting slug to “<stack_slug>” will make the instance
available at “https://<stack_slug>.grafana.net".`,
				ValidateFunc: validation.StringMatch(stackSlugRegex, "must be a lowercase alphanumeric string and must start with a letter."),
			},
			"region_slug": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: `Region slug to assign to this stack.
Changing region will destroy the existing stack and create a new one in the desired region`,
				ValidateFunc: validation.StringInSlice([]string{"au", "eu", "us"}, false),
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "Custom URL for the Grafana instance. Must have a CNAME setup to point to `.grafana.net` before creating the stack",
			},
			"wait_for_readiness": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether to wait for readiness of the stack after creating it. The check is a HEAD request to the stack URL (Grafana instance).",
			},
			"org_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Organization id to assign to this stack.",
			},
			"org_slug": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Organization slug to assign to this stack.",
			},
			"org_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Organization name to assign to this stack.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the stack.",
			},
			"prometheus_user_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Promehteus user ID. Used for e.g. remote_write.",
			},
			"prometheus_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Prometheus url for this instance.",
			},
			"prometheus_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Prometheus name for this instance.",
			},
			"prometheus_remote_endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Use this URL to query hosted metrics data e.g. Prometheus data source in Grafana",
			},
			"prometheus_remote_write_endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Use this URL to send prometheus metrics to Grafana cloud",
			},
			"prometheus_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Prometheus status for this instance.",
			},
			"alertmanager_user_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "User ID of the Alertmanager instance configured for this stack.",
			},
			"alertmanager_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the Alertmanager instance configured for this stack.",
			},
			"alertmanager_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Base URL of the Alertmanager instance configured for this stack.",
			},
			"alertmanager_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the Alertmanager instance configured for this stack.",
			},
			"logs_user_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"logs_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"logs_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"logs_status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func CreateStack(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gcloudapi

	stack := &gapi.CreateStackInput{
		Name:   d.Get("name").(string),
		Slug:   d.Get("slug").(string),
		URL:    d.Get("url").(string),
		Region: d.Get("region_slug").(string),
	}

	stackID, err := client.NewStack(stack)
	if err != nil && err.Error() == "409 Conflict" {
		return diag.Errorf("Error: A Grafana stack with the name '%s' already exists.", stack.Name)
	}

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(stackID, 10))

	if diag := ReadStack(ctx, d, meta); diag != nil {
		return diag
	}

	return waitForStackReadiness(ctx, d)
}

func UpdateStack(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gcloudapi
	stackID, _ := strconv.ParseInt(d.Id(), 10, 64)

	// The underlying API olnly allows to update the name and description.
	allowedChanges := []string{"name", "description", "slug"}
	if d.HasChangesExcept(allowedChanges...) {
		return diag.Errorf("Error: Only name, slug and description can be updated.")
	}

	if d.HasChange("name") || d.HasChange("description") || d.HasChanges("slug") {
		stack := &gapi.UpdateStackInput{
			Name:        d.Get("name").(string),
			Slug:        d.Get("slug").(string),
			Description: d.Get("description").(string),
		}
		err := client.UpdateStack(stackID, stack)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if diag := ReadStack(ctx, d, meta); diag != nil {
		return diag
	}

	return waitForStackReadiness(ctx, d)
}

func DeleteStack(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gcloudapi
	slug := d.Get("slug").(string)
	if err := client.DeleteStack(slug); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func ReadStack(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gcloudapi

	idStr := d.Id()
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.Errorf("Invalid id: %#v", idStr)
	}

	stack, err := client.StackByID(id)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing stack %s from state because it no longer exists in grafana", d.Get("name").(string))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if stack.Status == "deleted" {
		log.Printf("[WARN] removing stack %s from state because it was deleted outside of Terraform", stack.Name)
		d.SetId("")
		return nil
	}

	FlattenStack(d, stack)

	return nil
}

func FlattenStack(d *schema.ResourceData, stack gapi.Stack) {
	id := strconv.FormatInt(stack.ID, 10)

	d.SetId(id)
	d.Set("name", stack.Name)
	d.Set("slug", stack.Slug)
	d.Set("url", stack.URL)
	d.Set("status", stack.Status)
	d.Set("region_slug", stack.RegionSlug)
	d.Set("description", stack.Description)

	d.Set("org_id", stack.OrgID)
	d.Set("org_slug", stack.OrgSlug)
	d.Set("org_name", stack.OrgName)

	d.Set("prometheus_user_id", stack.HmInstancePromID)
	d.Set("prometheus_url", stack.HmInstancePromURL)
	d.Set("prometheus_name", stack.HmInstancePromName)
	d.Set("prometheus_remote_endpoint", path.Join(stack.HmInstancePromURL, "api/prom"))
	d.Set("prometheus_remote_write_endpoint", path.Join(stack.HmInstancePromURL, "api/prom/push"))
	d.Set("prometheus_status", stack.HmInstancePromStatus)

	d.Set("logs_user_id", stack.HlInstanceID)
	d.Set("logs_url", stack.HlInstanceURL)
	d.Set("logs_name", stack.HlInstanceName)
	d.Set("logs_status", stack.HlInstanceStatus)

	d.Set("alertmanager_user_id", stack.AmInstanceID)
	d.Set("alertmanager_name", stack.AmInstanceName)
	d.Set("alertmanager_url", stack.AmInstanceURL)
	d.Set("alertmanager_status", stack.AmInstanceStatus)
}

// waitForStackReadiness retries until the stack is ready, verified by querying the Grafana URL
func waitForStackReadiness(ctx context.Context, d *schema.ResourceData) diag.Diagnostics {
	if wait := d.Get("wait_for_readiness").(bool); !wait {
		return nil
	}

	err := resource.RetryContext(ctx, 2*time.Minute, func() *resource.RetryError {
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, d.Get("url").(string), nil)
		if err != nil {
			return resource.NonRetryableError(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return resource.NonRetryableError(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return resource.RetryableError(errors.New("stack is not ready yet"))
		}

		return nil
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("error waiting for stack to be ready: %w", err))
	}

	return nil
}
