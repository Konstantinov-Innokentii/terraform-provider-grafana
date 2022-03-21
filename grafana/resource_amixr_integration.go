package grafana

import (
	aapi "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"log"
)

var integrationTypes = []string{
	"grafana",
	"webhook",
	"alertmanager",
	"kapacitor",
	"fabric",
	"newrelic",
	"datadog",
	"pagerduty",
	"pingdom",
	"elastalert",
	"amazon_sns",
	"curler",
	"sentry",
	"formatted_webhook",
	"heartbeat",
	"demo",
	"manual",
	"stackdriver",
	"uptimerobot",
	"sentry_platform",
	"zabbix",
	"prtg",
	"slack_channel",
	"inbound_email",
}

func ResourceAmixrIntegration() *schema.Resource {
	return &schema.Resource{
		Create: resourceIntegrationCreate,
		Read:   resourceIntegrationRead,
		Update: resourceIntegrationUpdate,
		Delete: resourceIntegrationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"team_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice(integrationTypes, false),
				ForceNew:     true,
			},
			"default_route": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"escalation_chain_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"slack": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"channel_id": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
							MaxItems: 1,
						},
					},
				},
				MaxItems: 1,
			},
			"link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"templates": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"resolve_signal": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"grouping_key": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"slack": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"title": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"message": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"image_url": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
							MaxItems: 1,
						},
					},
				},
				MaxItems: 1,
			},
		},
	}
}

func resourceIntegrationCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*aapi.Client)

	teamIdData := d.Get("team_id").(string)
	nameData := d.Get("name").(string)
	typeData := d.Get("type").(string)
	templatesData := d.Get("templates").([]interface{})

	createOptions := &aapi.CreateIntegrationOptions{
		TeamId:    teamIdData,
		Name:      nameData,
		Type:      typeData,
		Templates: expandTemplates(templatesData),
	}

	integration, _, err := client.Integrations.CreateIntegration(createOptions)
	if err != nil {
		return err
	}

	d.SetId(integration.ID)

	return resourceIntegrationRead(d, m)
}

func resourceIntegrationUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*aapi.Client)

	nameData := d.Get("name").(string)
	templateData := d.Get("templates").([]interface{})
	defaultRouteData := d.Get("default_route").([]interface{})

	updateOptions := &aapi.UpdateIntegrationOptions{
		Name:         nameData,
		Templates:    expandTemplates(templateData),
		DefaultRoute: expandDefaultRoute(defaultRouteData),
	}

	integration, _, err := client.Integrations.UpdateIntegration(d.Id(), updateOptions)
	if err != nil {
		return err
	}

	d.SetId(integration.ID)

	return resourceIntegrationRead(d, m)
}

func resourceIntegrationRead(d *schema.ResourceData, m interface{}) error {

	client := m.(*aapi.Client)
	options := &aapi.GetIntegrationOptions{}
	integration, _, err := client.Integrations.GetIntegration(d.Id(), options)
	if err != nil {
		return err
	}

	d.Set("team_id", integration.TeamId)
	d.Set("default_route", flattenDefaultRoute(integration.DefaultRoute))
	d.Set("name", integration.Name)
	d.Set("type", integration.Type)
	d.Set("templates", flattenTemplates(integration.Templates))
	d.Set("link", integration.Link)

	return nil
}

func resourceIntegrationDelete(d *schema.ResourceData, m interface{}) error {
	log.Printf("[DEBUG] delete amixr integration")

	client := m.(*aapi.Client)
	options := &aapi.DeleteIntegrationOptions{}
	_, err := client.Integrations.DeleteIntegration(d.Id(), options)
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func flattenRouteSlack(in *aapi.SlackRoute) []map[string]interface{} {
	slack := make([]map[string]interface{}, 0, 1)

	out := make(map[string]interface{})

	if in.ChannelId != nil {
		out["channel_id"] = in.ChannelId
		slack = append(slack, out)
	}
	return slack
}

func expandRouteSlack(in []interface{}) *aapi.SlackRoute {
	slackRoute := aapi.SlackRoute{}

	for _, r := range in {
		inputMap := r.(map[string]interface{})
		if inputMap["channel_id"] != "" {
			channelId := inputMap["channel_id"].(string)
			slackRoute.ChannelId = &channelId
		}
	}

	return &slackRoute

}

func flattenTemplates(in *aapi.Templates) []map[string]interface{} {
	templates := make([]map[string]interface{}, 0, 1)
	out := make(map[string]interface{})

	out["grouping_key"] = in.GroupingKey
	out["resolve_signal"] = in.ResolveSignal
	out["slack"] = flattenSlackTemplate(in.Slack)

	add := false

	if in.GroupingKey != nil {
		out["grouping_key"] = in.GroupingKey
		add = true
	}
	if in.ResolveSignal != nil {
		out["resolve_signal"] = in.ResolveSignal
		add = true

	}
	if in.Slack != nil {
		flattenSlackTemplate := flattenSlackTemplate(in.Slack)
		if len(flattenSlackTemplate) > 0 {
			out["resolve_signal"] = in.ResolveSignal
			add = true
		}
	}

	if add {
		templates = append(templates, out)
	}

	return templates
}

func flattenSlackTemplate(in *aapi.SlackTemplate) []map[string]interface{} {
	slackTemplates := make([]map[string]interface{}, 0, 1)

	add := false

	slackTemplate := make(map[string]interface{})

	if in.Title != nil {
		slackTemplate["title"] = in.Title
		add = true
	}
	if in.ImageURL != nil {
		slackTemplate["image_url"] = in.ImageURL
		add = true
	}
	if in.Message != nil {
		slackTemplate["message"] = in.Message
		add = true
	}

	if add {
		slackTemplates = append(slackTemplates, slackTemplate)
	}

	return slackTemplates
}

func expandTemplates(input []interface{}) *aapi.Templates {

	templates := aapi.Templates{}

	for _, r := range input {
		inputMap := r.(map[string]interface{})
		if inputMap["grouping_key"] != "" {
			gk := inputMap["grouping_key"].(string)
			templates.GroupingKey = &gk
		}
		if inputMap["resolve_signal"] != "" {
			rs := inputMap["resolve_signal"].(string)
			templates.ResolveSignal = &rs
		}
		if inputMap["slack"] == nil {
			templates.Slack = nil
		} else {
			templates.Slack = expandSlackTemplate(inputMap["slack"].([]interface{}))
		}
	}
	return &templates
}

func expandSlackTemplate(in []interface{}) *aapi.SlackTemplate {

	slackTemplate := aapi.SlackTemplate{}
	for _, r := range in {
		inputMap := r.(map[string]interface{})
		if inputMap["title"] != "" {
			t := inputMap["title"].(string)
			slackTemplate.Title = &t
		}
		if inputMap["message"] != "" {
			m := inputMap["message"].(string)
			slackTemplate.Message = &m
		}
		if inputMap["image_url"] != "" {
			iu := inputMap["image_url"].(string)
			slackTemplate.ImageURL = &iu
		}
	}
	return &slackTemplate
}

func flattenDefaultRoute(in *aapi.DefaultRoute) []map[string]interface{} {
	defaultRoute := make([]map[string]interface{}, 0, 1)
	out := make(map[string]interface{})
	out["id"] = in.ID
	out["escalation_chain_id"] = in.EscalationChainId
	out["slack"] = flattenRouteSlack(in.SlackRoute)

	defaultRoute = append(defaultRoute, out)
	return defaultRoute
}

func expandDefaultRoute(input []interface{}) *aapi.DefaultRoute {
	defaultRoute := aapi.DefaultRoute{}

	for _, r := range input {
		inputMap := r.(map[string]interface{})
		id := inputMap["id"].(string)
		defaultRoute.ID = id
		if inputMap["escalation_chain_id"] != "" {
			escalation_chain_id := inputMap["escalation_chain_id"].(string)
			defaultRoute.EscalationChainId = &escalation_chain_id
		}
		if inputMap["slack"] == nil {
			defaultRoute.SlackRoute = nil
		} else {
			defaultRoute.SlackRoute = expandRouteSlack(inputMap["slack"].([]interface{}))
		}
	}
	return &defaultRoute
}
