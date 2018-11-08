package vsphere

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

func dataSourceVsphereJpmcHost() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVsphereJpmcHostRead,

		Schema: map[string]*schema.Schema{
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
		},
	}
}

func dataSourceVsphereJpmcHostRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	name := d.Get("name").(string)
	dcID := d.Get("datacenter_id").(string)
	dc, err := HostDatacenterFromID(client, dcID)
	if err != nil {
		return fmt.Errorf("error fetching datacenter: %s", err)
	}
	hs, err := HostSystemOrDefault(client, name, dc)
	if err != nil {
		return fmt.Errorf("error fetching host: %s", err)
	}
	rp, err := HostResourcePool(hs)
	if err != nil {
		return err
	}
	err = d.Set("resource_pool_id", rp.Reference().Value)
	if err != nil {
		return err
	}
	id := hs.Reference().Value
	d.SetId(id)
	return nil
}

func HostDatacenterFromID(client *govmomi.Client, id string) (*object.Datacenter, error) {
	finder := find.NewFinder(client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  "Datacenter",
		Value: id,
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	ds, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("could not find datacenter with id: %s: %s", id, err)
	}
	return ds.(*object.Datacenter), nil
}
