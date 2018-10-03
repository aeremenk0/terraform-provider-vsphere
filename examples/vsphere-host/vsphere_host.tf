provider "vsphere" {
	user="administrator@vsphere.local"
	password="VMware1!"
	vsphere_server="192.168.75.100"
	allow_unverified_ssl=true
}


data "vsphere_datacenter" "dc"{
	name = "Datacenter"
}

variable "config1" {
	type="map"
	default = {
		default_ip="192.168.75.135"
		default_username="root"
		default_password="VMware1!"
		hostname="192.168.75.135"
		fqdn="asdf"
		dns="asdf"
		root_password="VMware1!"
		ntp_server="asdf"
		enable_ssh=true
		connected=true
		iscsi_adapter="asdf"
	}
}

resource "vsphere_host" "h1"{
	name = "192.168.75.135"
	datacenter_id = "${data.vsphere_datacenter.dc.id}"
	host_config = "${var.config1}"
	
}


r := strings.NewReader(s)
req, err := http.NewRequest("POST", url, r)
req.Header.Add("vmware-api-session-id", apiSessionId)
res, err := c.Do(req)