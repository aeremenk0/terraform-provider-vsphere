package vsphere

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/hostsystem"
	"github.com/vmware/govmomi/govc/host/esxcli"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

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
		"advanced_options": &schema.Schema{
			Type:        schema.TypeMap,
			Description: "Configuration for the host's advanced options.",
			Optional:    true,
		},
		"host_id": &schema.Schema{
			Type:        schema.TypeString,
			Description: "The managed object ID of the host's root resource pool.",
			Computed:    true,
		},
		"service_policy": &schema.Schema{
			Type:        schema.TypeMap,
			Description: "Configuration for the host's service policy.",
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

type errorCheck func(response *http.Response) error

func defaultErrorCheck(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 303 {
		return fmt.Errorf("Unexpected response code '%d'", response.StatusCode)
	}

	return nil
}

func connectErrorCheck(response *http.Response) error {
	if response.StatusCode != 400 && (response.StatusCode < 200 || response.StatusCode >= 303) {
		return fmt.Errorf("Unexpected response code '%d'", response.StatusCode)
	}

	return nil
}

func doWithCheck(client *VSphereClient, method string, resource string, body io.Reader,
	result interface{}, check errorCheck) error {

	url := "https://" + client.vimClient.URL().Host + resource
	req, err := http.NewRequest(method, url, body)

	if err != nil {
		return err
	}

	c := client.tagsClient
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("vmware-api-session-id", c.SessionID())

	res, err := c.HTTP.Do(req)

	if err != nil {
		return err
	}

	// Get the ID
	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return err
	}

	if err := check(res); err != nil {
		return fmt.Errorf(err.Error()+": %v", string(data))
	}

	if result != nil {
		if err := json.Unmarshal(data, &result); err != nil {
			return fmt.Errorf("Response is not json, %v", string(data))
		}
	}

	return nil
}

func do(client *VSphereClient, method string, resource string, body io.Reader, result interface{}) error {
	return doWithCheck(client, method, resource, body, result, defaultErrorCheck)
}

type HostData struct {
	Folder                 string `json:"folder"`
	HostName               string `json:"hostname"`
	Password               string `json:"password"`
	Username               string `json:"user_name"`
	ThumbprintVerification string `json:"thumbprint_verification"`
}

type HostSpec struct {
	Spec HostData `json:"spec"`
}

func NewHostSpec(folder string, hostname string, password string, username string) *HostSpec {
	return &HostSpec{HostData{
		Folder:                 folder,
		HostName:               hostname,
		Password:               password,
		Username:               username,
		ThumbprintVerification: "NONE",
	}}
}

func s2json(s interface{}) io.Reader {
	b := new(bytes.Buffer)

	if s != nil {
		json.NewEncoder(b).Encode(s)
	}

	return b
}

func getFolder(vsClient *VSphereClient) (string, error) {
	resp := make(map[string]interface{})

	err := do(vsClient, "GET", "/rest/vcenter/folder", strings.NewReader(""), &resp)

	if err != nil {
		return "", err
	}

	array := resp["value"].([]interface{})

	folder := ""
	for i := range array {
		if array[i].(map[string]interface{})["type"].(string) == "HOST" {
			folder = array[i].(map[string]interface{})["folder"].(string)
		}
	}

	return folder, nil
}

func createHost(vsClient *VSphereClient, hostSpec HostSpec) (string, error) {
	r := make(map[string]interface{})

	if err := do(vsClient, "POST", "/rest/vcenter/host", s2json(hostSpec), &r); err != nil {
		return "", err
	}

	hostID := r["value"].(string)

	return hostID, nil
}

func connectHost(vsClient *VSphereClient, hostID string) error {
	if err := doWithCheck(vsClient, "POST", "/rest/vcenter/host/"+hostID+"/connect",
		strings.NewReader(""), nil, connectErrorCheck); err != nil {
		return err
	}

	return nil
}

func disconnectHost(vsClient *VSphereClient, hostID string) error {
	if err := doWithCheck(vsClient, "POST", "/rest/vcenter/host/"+hostID+"/disconnect",
		strings.NewReader(""), nil, connectErrorCheck); err != nil {
		return err
	}

	return nil
}

func deleteHost(vsClient *VSphereClient, hostID string) error {
	if err := do(vsClient, "DELETE", "/rest/vcenter/host/"+hostID, strings.NewReader(""), nil); err != nil {
		return err
	}

	return nil
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

	vsClient := meta.(*VSphereClient)

	// Get the parameters for the API call
	config := d.Get("host_config").(map[string]interface{})

	hostname := d.Get("name").(string)
	username := "root"

	var password string
	var connected string

	// Try to get the connection credentials from the default fields.
	// These fields are intended for initial login.
	// Terraform will set the username and password to whatever is specified in the username and password fields,
	// and then use those fields to login afterwards.  Same thing with the host name.

	if val, ok := config["root_password"]; ok {
		password = val.(string)
	}

	if val, ok := config["connected"]; ok {
		connected = val.(string)
	}

	folder, err := getFolder(vsClient)

	if err != nil {
		return err
	}

	// API Call to add the host
	hostSpec := NewHostSpec(folder, hostname, password, username)
	hostID, err := createHost(vsClient, *hostSpec)

	if err != nil {
		return err
	}

	d.Set("host_id", hostID)

	// Set ISCSI
	// Need:
	// enable
	//
	// adapter_name
	// chap_name
	// chap_secret
	// chap_direction - set to mutual for now
	// send_target - will support only one for now

	/*

			example:
			variable "iscsi_config1" {
			type="map"
			default = {
				enable = true
				adapter_name="vmhba65"
				chap_name="test"
				chap_secret="testabc"
				chap_direction="mutual"
				chap_level="required"
				send_target="192.168.100.1:443"
			}
		}

	*/

	iscsiConfig := d.Get("iscsi_config").(map[string]interface{})

	// iscsi software set -e true
	// iscsi adapter list
	// iscsi adapter auth chap set -A adapter_name -N chap_name -S chap_secret -l required
	// iscsi adapter auth chap set -A adapter_name -N chap_name -S chap_secret -l required -d mutual
	// iscsi adapter discovry sendtarget add -A adapter_name -a ip:port

	// iscsi
	if val, ok := iscsiConfig["enable"]; ok {
		if val.(string) == "1" {
			argsIscsi := []string{"iscsi", "software", "set", "-e", "true"}
			// send to esx cli
			err = runEsxCliCommand(d, meta, argsIscsi)
			if err != nil {
				return err
			}

			// get default adapter name
			/*
				argsIscsiDef := []string{"iscsi", "adapter", "list", "|", "grep","iscsi_vmk","|","awk","'{print $1;}'"}
				// send to esx cli
				default_adapter, err := runEsxCliCommandOutput(d, meta, argsIscsiDef)
				if err != nil {
					return err
				}


				// set iscsi adapter name
				// WIP
				if default_adapter != "" {
					if val, ok := iscsiConfig["adapter_name"]; ok {
						_ = val
						argsIscsi := []string{"iscsi", "adapter", "set", "-A", default_adapter, "-N", iscsiConfig["adapter_name"].(string), "-a", iscsiConfig["adapter_name"].(string)}
						// send to esx cli
						err = runEsxCliCommand(d, meta, argsIscsi)
						if err != nil {
							return err
						}
					}
				}
			*/

			// Set the ISCSI Chap config
			err = validateIscsiChapInputs(iscsiConfig)
			if err == nil {
				argsIscsiChap := []string{"iscsi", "adapter", "auth", "chap", "set", "-A", iscsiConfig["adapter_name"].(string), "-N", iscsiConfig["chap_name"].(string), "-S", iscsiConfig["chap_secret"].(string), "-l", iscsiConfig["chap_level"].(string)}
				// send to esx cli
				err = runEsxCliCommand(d, meta, argsIscsiChap)
				if err != nil {
					return err
				}

				// Set to mutual
				if val, ok := iscsiConfig["chap_direction"]; ok {
					argsIscsiChap = []string{"iscsi", "adapter", "auth", "chap", "set", "-A", iscsiConfig["adapter_name"].(string), "-N", iscsiConfig["chap_name"].(string), "-S", iscsiConfig["chap_secret"].(string), "-l", iscsiConfig["chap_level"].(string), "-d", val.(string)}
					// send to esx cli
					err = runEsxCliCommand(d, meta, argsIscsiChap)
					if err != nil {
						return err
					}
				}

			}

			// iSCSI set target
			// WIP
			err = validateIscsiSendTargetInputs(iscsiConfig)
			if err == nil {
				argsIscsiTarget := []string{"iscsi", "adapter", "discovery", "sendtarget", "add", "-A", iscsiConfig["adapter_name"].(string), "-a", iscsiConfig["send_target"].(string)}
				// send to esx cli
				err = runEsxCliCommand(d, meta, argsIscsiTarget)
				if err != nil {
					return err
				}
			}

			// Set the nics for iscsi
			// Still need to do this part

		} else {
			argsIscsi := []string{"iscsi", "software", "set", "-e", "false"}
			// send to esx cli
			err = runEsxCliCommand(d, meta, argsIscsi)
			if err != nil {
				return err
			}
		}
	}

	// Set Maintenance Mode
	if val, ok := config["maintenance_mode"]; ok {
		var argsMaint []string
		if val.(string) == "1" {
			argsMaint = []string{"system", "maintenanceMode", "set", "-e", "true"}
		} else if val.(string) == "0" {
			argsMaint = []string{"system", "maintenanceMode", "set", "-e", "false"}
		}
		// send to esx cli
		err = runEsxCliCommand(d, meta, argsMaint)
		_ = err
	}

	// Set NTP
	if val, ok := config["ntp_server"]; ok {
		_ = val
		argsNtp := []string{}
		_ = argsNtp
	}
	// Set networking information
	// WIP
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
	// WIP
	if val, ok := config["enable_ssh"]; ok {
		_ = val
		argsSsh := []string{}
		_ = argsSsh
	}

	// Set default username and password
	// WIP
	if val, ok := config["root_password"]; ok {
		_ = val
		argsRpw := []string{"system", "account", "set", "--id", "root", "--password", val.(string), "--password-confirmation", val.(string)}
		_ = argsRpw
	}

	// Set advanced options
	// come back to this
	if val, ok := config["advanced_options"]; ok {
		_ = val
		argsAdv := []string{"system", "settings", "advanced", "set", "-o", "", "-s", ""}

		values := config["advanced_options"].(map[string]string)

		for k := range values {
			// Need to check if the arguement is supposed to be a string or an integer
			if _, err := fmt.Sscanf(values[k], "%d", &k); err == nil {
				argsAdv = []string{"system", "settings", "advanced", "set", "-o", k, "-i", values[k]}
			} else {
				argsAdv = []string{"system", "settings", "advanced", "set", "-o", k, "-s", values[k]}
			}
			err = runEsxCliCommand(d, meta, argsAdv)
			if err != nil {
				return err
			}
		}
	}

	// Set whether the host is connected or not
	if connected == "1" {
		if err := connectHost(vsClient, hostID); err != nil {
			return err
		}
	} else if connected == "0" {
		if err := disconnectHost(vsClient, hostID); err != nil {
			return err
		}
	}

	servicePolicy := d.Get("service_policy").(map[string]interface{})

	c := meta.(*VSphereClient).vimClient
	dcID := d.Get("datacenter_id").(string)
	dc, err := datacenterFromID(c, dcID)

	hs, err := hostsystem.SystemOrDefault(c, hostname, dc)

	ctx := context.Background()
	hostConfigManager := hs.ConfigManager()
	hostServiceSystem, err := hostConfigManager.ServiceSystem(ctx)

	for serviceName, flag := range servicePolicy {
		if err := hostServiceSystem.UpdatePolicy(ctx, serviceName, flag.(string)); err != nil {
			return err
		}
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
	vsClient := meta.(*VSphereClient)

	if err := deleteHost(vsClient, d.Get("host_id").(string)); err != nil {
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

func runEsxCliCommandOutput(d *schema.ResourceData, meta interface{}, args []string) (string, error) {

	c := meta.(*VSphereClient).vimClient

	name := d.Get("name").(string)
	dcID := d.Get("datacenter_id").(string)
	dc, err := datacenterFromID(c, dcID)

	if err != nil {
		return "", fmt.Errorf("error fetching datacenter: %s", err)
	}

	hs, err := hostsystem.SystemOrDefault(c, name, dc)

	//arg := []string{"system","maintenanceMode","set","-e","true"}
	//com := esxcli.NewCommand(arg)

	exec, err := esxcli.NewExecutor(c.Client, hs)
	if err != nil {
		return "", err
	}

	resp, err := exec.Run(args)

	_ = resp

	return "", err

}

func validateIscsiChapInputs(params map[string]interface{}) error {
	if val, ok := params["adapter_name"]; ok {
		_ = val
	} else {
		return fmt.Errorf("Iscsi parameter \"adapter_name\" is undefined")
	}

	if val, ok := params["chap_name"]; ok {
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

func getKeys(mymap map[string]string) []string {
	keys := make([]string, 0, len(mymap))
	for k := range mymap {
		keys = append(keys, k)
	}
	return keys
}
