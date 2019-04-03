package vsphere

import "github.com/hashicorp/terraform/helper/schema"

func dataSourceVSphereHostConfig() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVSphereDatastoreRead,

		Schema: map[string]*schema.Schema{
			"default_ip": {
				Type:        schema.TypeString,
				Description: "Used for initial connection to the host.",
				Optional:    true,
			},
			"default_username": {
				Type:        schema.TypeString,
				Description: "Used for initial connection to the host.",
				Optional:    true,
			},
			"default_password": {
				Type:        schema.TypeString,
				Description: "Used for initial connection to the host.",
				Optional:    true,
			},
			"hostname": {
				Type:        schema.TypeString,
				Description: "",
				Required:    true,
			},
			"fqdn": {
				Type:        schema.TypeString,
				Description: "",
				Required:    true,
			},
			"dns": {
				Type:        schema.TypeString,
				Description: "",
				Required:    true,
			},
			"root_password": {
				Type:        schema.TypeString,
				Description: "",
				Optional:    true,
			},
			"ntp_server": {
				Type:        schema.TypeString,
				Description: "",
				Optional:    true,
			},
			"enable_ssh": {
				Type:        schema.TypeBool,
				Description: "",
				Required:    true,
			},
			"connected": {
				Type:        schema.TypeBool,
				Description: "",
				Required:    true,
			},
			"iscsi_adapter": {
				Type:        schema.TypeString,
				Description: "",
				Required:    true,
			},
		},
	}
}
