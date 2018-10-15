package vsphere

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/hostsystem"
	"github.com/vmware/govmomi/govc/host/esxcli"
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
		"iscsi_config": &schema.Schema{
			Type:        schema.TypeMap,
			Description: "Configuration for the host iscsi adapter.",
			Optional:    true,
		},
		"host_id": &schema.Schema{
			Type:        schema.TypeString,
			Description: "The managed object ID of the host's root resource pool.",
			Computed:    true,
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

	// Get REST Client for Session ID
	rc := meta.(*VSphereClient).tagsClient

	apiSessionId := rc.SessionID()

	// Get the parameters for the API call
	config := d.Get("host_config").(map[string]interface{})

	hostname := d.Get("name").(string)
	username := "root"
	var password string
	var connected string

	// Try to get the connection credentials from the default fields.
	// These fields are intended for initial login.  Terraform will set the username and password to whatever is specified in the username and password fields, and then use those fields to login afterwards.  Same thing with the host name.

	if val, ok := config["root_password"]; ok {
		password = val.(string)
	}

	if username == "" {
		username = "root"
	}
	if password == "" {
		password = "VMware1!"
	}

	if val, ok := config["connected"]; ok {
		connected = val.(string)
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
	// Check that the request was successful
	var host_id string
	if val, ok := contout["type"]; ok {
		if val.(string) == "com.vmware.vapi.std.errors.internal_server_error" {
			// Host has already been added
			req, err = http.NewRequest("GET", url, strings.NewReader(""))
			req.Header.Add("Accept", "application/json")
			req.Header.Add("Content-Type", "application/json")

			// need a separate call to get session id
			req.Header.Add("vmware-api-session-id", apiSessionId)
			res, err = c.Do(req)
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return err
			}
			contout := make(map[string]interface{})
			err = json.Unmarshal(body, &contout)

			array := contout["value"].([]interface{})
			for i := range array {

				if array[i].(map[string]interface{})["name"].(string) == hostname {
					host_id = array[i].(map[string]interface{})["host"].(string)
				}
			}
		}
	} else {
		host_id = contout["value"].(string)
	}
	d.Set("host_id", host_id)

	// Set ISCSI
	// Need:
	// adapter_name
	// auth_name
	// chap_secret
	// direction - set to mutual for now
	// send_target - will support only one for now

	iscsiConfig := d.Get("iscsi_config").(map[string]interface{})
	// set iscsi adapter
	if val, ok := iscsiConfig["adapter_name"]; ok {
		_ = val
		argsIscsi := []string{"iscsi", "adapter", "set", "-A", iscsiConfig["adapter_name"].(string)}
		// send to esx cli
		err = runEsxCliCommand(d, meta, argsIscsi)
		if err != nil {
			return err
		}
	}

	err = validateIscsiChapInputs(iscsiConfig)
	if err == nil {
		argsIscsiChap := []string{"iscsi", "adapter", "auth", "chap", "set", "-A", iscsiConfig["adapter_name"].(string), "-N", iscsiConfig["auth_name"].(string), "-S", iscsiConfig["chap_secret"].(string)}
		// send to esx cli
		err = runEsxCliCommand(d, meta, argsIscsiChap)
		if err != nil {
			return err
		}
	}

	// iSCSI set target
	err = validateIscsiSendTargetInputs(iscsiConfig)
	if err == nil {
		argsIscsiTarget := []string{"iscsi", "adapter", "auth", "chap", "set", "-A", iscsiConfig["adapter_name"].(string), "-a", iscsiConfig["send_target"].(string)}
		// send to esx cli
		err = runEsxCliCommand(d, meta, argsIscsiTarget)
		if err != nil {
			return err
		}
	}

	// Set Maintenance Mode
	/*
		if val, ok := config["maintenance_mode"]; ok {
			var argsMaint []string
			if val.(string) == "1" {
				argsMaint = []string{"system", "maintenanceMode", "set", "-e", "true"}
			} else if val.(string) == "0" {
				argsMaint = []string{"system", "maintenanceMode", "set", "-e", "false"}
			}
			// send to esx cli
			err = runEsxCliCommand(d, meta, argsMaint)
			if err != nil {
				return err
			}
		}
	*/

	// Set NTP
	if val, ok := config["ntp_server"]; ok {
		_ = val
		argsNtp := []string{}
		_ = argsNtp
	}
	// Set networking information
	if val, ok := config["dns"]; ok {
		if val.(string) == "dhcp" {
			argsDns := []string{"network", "ip", "dns"}
			_ = argsDns
		} else {
			argsDns := []string{}
			_ = argsDns
		}

	}

	// Set whether SSH is enabled
	if val, ok := config["enable_ssh"]; ok {
		_ = val
		argsSsh := []string{}
		_ = argsSsh
	}

	// Set default username and password
	/*
		if val, ok := config["root_password"]; ok {
			_ = val
			argsRpw := []string{"system", "account", "set", "--id", "root", "--password", val.(string), "--password-confirmation", val.(string)}
			_ = argsRpw
		}
	*/

	// Set whether the host is connected or not
	if connected == "1" {
		urlcon := "https://" + c.URL().Host + "/rest/vcenter/host/" + host_id + "/connect"
		r := strings.NewReader("")
		req, err := http.NewRequest("POST", urlcon, r)
		if err != nil {
			return err
		}
		req.Header.Add("vmware-api-session-id", apiSessionId)
		rescon, err := c.Do(req)
		_ = rescon
	} else if connected == "0" {
		urlcon := "https://" + c.URL().Host + "/rest/vcenter/host/" + host_id + "/disconnect"
		r := strings.NewReader("")
		req, err := http.NewRequest("POST", urlcon, r)
		if err != nil {
			return err
		}
		req.Header.Add("vmware-api-session-id", apiSessionId)
		rescon, err := c.Do(req)
		_ = rescon
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

	// Get REST Client for Session ID
	rc := meta.(*VSphereClient).tagsClient

	apiSessionId := rc.SessionID()

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

func runEsxCliCommand(d *schema.ResourceData, meta interface{}, args []string) error {

	c := meta.(*VSphereClient).vimClient

	name := d.Get("name").(string)
	dcID := d.Get("datacenter_id").(string)
	dc, err := datacenterFromID(c, dcID)

	if err != nil {
		return fmt.Errorf("error fetching datacenter: %s", err)
	}

	hs, err := hostsystem.SystemOrDefault(c, name, dc)

	//arg := []string{"system","maintenanceMode","set","-e","true"}
	//com := esxcli.NewCommand(arg)

	exec, err := esxcli.NewExecutor(c.Client, hs)
	if err != nil {
		return err
	}

	resp, err := exec.Run(args)

	_ = resp

	return err

}

func validateIscsiChapInputs(params map[string]interface{}) error {
	if val, ok := params["adapter_name"]; ok {
		_ = val
	} else {
		return fmt.Errorf("Iscsi parameter \"adapter_name\" is undefined")
	}

	if val, ok := params["auth_name"]; ok {
		_ = val
	} else {
		return fmt.Errorf("Iscsi parameter \"auth_name\" is undefined")
	}

	if val, ok := params["chap_secret"]; ok {
		_ = val
	} else {
		return fmt.Errorf("Iscsi parameter \"chap_secret\" is undefined")
	}

	return nil
}

func validateIscsiSendTargetInputs(params map[string]interface{}) error {
	if val, ok := params["adapter_name"]; ok {
		_ = val
	} else {
		return fmt.Errorf("Iscsi parameter \"adapter_name\" is undefined")
	}

	if val, ok := params["send_target"]; ok {
		_ = val
	} else {
		return fmt.Errorf("Iscsi parameter \"send_target\" is undefined")
	}

	return nil
}
