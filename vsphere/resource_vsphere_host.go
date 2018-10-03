package vsphere

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/hostsystem"
	"github.com/vmware/govmomi"
	"io/ioutil"
	"net/http"
	"strings"
)

type host struct {
	datacenter string
	connected  bool
	name       string
}

type itemdata []map[string]interface{}

func resourceVSphereHost() *schema.Resource {

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
		"host_config": &schema.Schema{
			Type:        schema.TypeMap,
			Description: "Configuration for the host.",
			Optional:    true,
		},
	}

	return &schema.Resource{
		Create: resourceVSphereHostCreate,
		Read:   resourceVSphereHostRead,
		Update: resourceVSphereHostUpdate,
		Delete: resourceVSphereHostDelete,
		Schema: s,
	}

}

func resourceVSphereHostCreate(d *schema.ResourceData, meta interface{}) error {

	// Watch out for this error: https://kb.vmware.com/s/article/2148065?lang=en_US

	// Add the iscsi piece into this - high priority
	// Add the ability to specify settings as a separate resource and pass that into the host
	//	- ISCSI
	// 	- NTP
	//	- DNS
	// 	- SSH
	// 	- IP Address
	// 	- Host name
	// 	- Set username and password for the host
	// add the ability to disconnect
	// add the ability to wipe the esx state on delete

	c := meta.(*VSphereClient).vimClient

	apiSessionId, err := getSessionId(c)
	if err != nil {
		return err
	}

	// Get the parameters for the API call
	config := d.Get("host_config").(map[string]interface{})

	var hostname string
	var username string
	var password string

	// Try to get the connection credentials from the default fields.
	// These fields are intended for initial login.  Terraform will set the username and password to whatever is specified in the username and password fields, and then use those fields to login afterwards.  Same thing with the host name.
	if val, ok := config["default_ip"]; ok {
		hostname = val.(string)
	} else if val, ok := config["fqdn"]; ok {
		hostname = val.(string)
	} else if val, ok := config["hostname"]; ok {
		hostname = val.(string)
	}

	if val, ok := config["default_username"]; ok {
		username = val.(string)
	} else {
		username = "root"
	}

	if val, ok := config["default_password"]; ok {
		password = val.(string)
	} else if val, ok := config["root_password"]; ok {
		username = val.(string)
	}

	if username == "" {
		username = "root"
	}
	if password == "" {
		password = "VMware1!"
	}

	// This will set the folder that the host is added to
	rf := strings.NewReader("")
	urlf := "https://" + c.URL().Host + "/rest/vcenter/folder"
	reqf, err := http.NewRequest("GET", urlf, rf)
	reqf.Header.Add("Accept", "application/json")
	reqf.Header.Add("Content-Type", "application/json")
	reqf.Header.Add("vmware-api-session-id", apiSessionId)
	resf, err := c.Do(reqf)
	// Get the ID
	bodyf, err := ioutil.ReadAll(resf.Body)
	contf := make(map[string]interface{})
	err = json.Unmarshal(bodyf, &contf)
	if err != nil {
		return err
	}

	array := contf["value"].([]interface{})

	// We should be getting the folder from the datacenter but we will deal with that later
	//dcID := d.Get("datacenter_id").(string)
	//dc, err := datacenterFromID(c, dcID)

	//folder := dc.InventoryPath

	folder := ""

	for i := range array {

		if array[i].(map[string]interface{})["type"].(string) == "HOST" {
			folder = array[i].(map[string]interface{})["folder"].(string)
		}
	}

	//folder := "group-h4"
	//,\"force_add\":\"true\"

	// API Call to add the host
	bod := "{\"spec\":{\"folder\":\"" + folder + "\",\"hostname\":\"" + hostname + "\",\"password\":\"" + password + "\",\"user_name\":\"" + username + "\",\"thumbprint_verification\":\"NONE\"}}"
	s := bod
	r := strings.NewReader(s)
	// Need to get part of the url from client
	url := "https://" + c.URL().Host + "/rest/vcenter/host"
	req, err := http.NewRequest("POST", url, r)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	// need a separate call to get session id
	req.Header.Add("vmware-api-session-id", apiSessionId)
	res, err := c.Do(req)

	if err != nil {
		return err
	}

	// Get the ID
	body, err := ioutil.ReadAll(res.Body)
	contout := make(map[string]interface{})
	err = json.Unmarshal(body, &contout)
	if err != nil {
		return err
	}
	//return fmt.Errorf("Response from vCenter: %s", contout)
	host_id := contout["value"]

	d.Set("host_id", host_id)

	//err = setIscsi(d,meta)

	// Set ISCSI
	if val, ok := config["iscsi_adapter"]; ok {
		_ = val
	}

	// Set NTP
	if val, ok := config["ntp_server"]; ok {
		_ = val
	}

	// Set networking information
	if val, ok := config["dns"]; ok {
		_ = val
	}

	// Set whether SSH is enabled
	if val, ok := config["enable_ssh"]; ok {
		_ = val
	}

	// Set default username and password
	if val, ok := config["root_username"]; ok {
		_ = val
	}

	if val, ok := config["root_password"]; ok {
		_ = val
	}

	// Set whether the host is connected or not
	if val, ok := config["connected"]; ok {
		_ = val
	}

	return resourceVSphereHostRead(d, meta)
}

func resourceVSphereHostRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	name := d.Get("name").(string)
	dcID := d.Get("datacenter_id").(string)
	dc, err := datacenterFromID(client, dcID)
	if err != nil {
		return fmt.Errorf("error fetching datacenter: %s", err)
	}
	hs, err := hostsystem.SystemOrDefault(client, name, dc)
	if err != nil {
		return fmt.Errorf("error fetching host in resourceVSphereHostRead: %s", err)
	}
	rp, err := hostsystem.ResourcePool(hs)
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

func resourceVSphereHostUpdate(d *schema.ResourceData, meta interface{}) error {
	// There isnt really much to do in order to update this stuff
	// try to add but dont do anything if there is an error?

	// maybe add abiity to disonnect host
	//

	err := resourceVSphereHostCreate(d, meta)

	return err
}

func resourceVSphereHostDelete(d *schema.ResourceData, meta interface{}) error {

	c := meta.(*VSphereClient).vimClient

	apiSessionId, err := getSessionId(c)
	if err != nil {
		return err
	}

	r := strings.NewReader("")
	// Need to get part of the url from client
	url := "https://" + c.URL().Host + "/rest/vcenter/host/" + d.Get("host_id").(string)
	req, err := http.NewRequest("DELETE", url, r)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	// need a separate call to get session id
	req.Header.Add("vmware-api-session-id", apiSessionId)
	res, err := c.Do(req)

	_ = res

	if err != nil {
		return err
	}

	return nil
}

func getSessionId(c *govmomi.Client) (string, error) {
	rauth := strings.NewReader("{}")

	urlauth := "https://" + c.URL().Host + "/rest/com/vmware/cis/session"
	reqauth, err := http.NewRequest("POST", urlauth, rauth)
	reqauth.Header.Add("Authorization", "Basic YWRtaW5pc3RyYXRvckB2c3BoZXJlLmxvY2FsOlZNd2FyZTEh")
	reqauth.Header.Add("Accept", "application/json")
	reqauth.Header.Add("Content-Type", "application/json")
	resauth, err := c.Do(reqauth)

	//out, err2 := ioutil.ReadAll(resauth.Body)
	//_ = err2
	//return fmt.Errorf("getting session id response: %s", out)
	if err != nil {
		return "", err
	}

	// Get the value from the body
	bodyauth, err := ioutil.ReadAll(resauth.Body)
	cont := make(map[string]interface{})
	err = json.Unmarshal(bodyauth, &cont)
	if err != nil {
		return "", err
	}
	apiSessionId := cont["value"].(string)
	return apiSessionId, nil
}

/*
func setIscsi(d *schema.ResourceData, meta interface{}) error{

	c2 := meta.(*VSphereClient).vimClient
	name := d.Get("name").(string)
	dcID := d.Get("datacenter_id").(string)
	dc, err := datacenterFromID(c2, dcID)

	if err != nil {
		return fmt.Errorf("error fetching datacenter: %s", err)
	}

	hs, err := hostsystem.SystemOrDefault(c2, name, dc)

	arg := []string{"system","maintenanceMode","set","-e","true"}
	//com := esxcli.NewCommand(arg)

	exec, err := esxcli.NewExecutor(c2.Client,hs)
	if err != nil{
		return err
	}

	resp, err := exec.Run(arg)

	_=resp


	return err

}
*/
