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
	}
}

variable "iscsi_config" {
	type="map"
	default = {
		adapter_name="test"
		auth_name="test"
		chap_secret="test"
		send_target="test"
	}
}

resource "vsphere_host" "h1"{
	name = "192.168.75.135"
	datacenter_id = "${data.vsphere_datacenter.dc.id}"
	host_config = "${var.config1}"
	iscsi_config = "${var.iscsi_config}"
	
}