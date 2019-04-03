package vsphere

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func schemaHostSpec() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"name": &schema.Schema{
			Type: schema.TypeString,
			Description: "The name of the host. This can be a name or path.	If not provided, the default host is used.",
			Optional: true,
		},
		"datacenter_id": &schema.Schema{
			Type:        schema.TypeString,
			Description: "The managed object ID of the datacenter to look for the host in.",
			Required:    true,
		},
		"resource_pool_id": &schema.Schema{
			Type:        schema.TypeString,
			Description: "The managed object ID of the host's root resource pool.",
			Computed:    true,
		},
	}

	return s
}
