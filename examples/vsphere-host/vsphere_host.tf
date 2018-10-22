data "vsphere_datacenter" "dc"{
	name = ""
}

variable "config1" {
	type="map"
	default = {
		dns=""
		root_password=""
		ntp_server=""
		enable_ssh=true
		connected=true
	}
}

variable "iscsi_config" {
	type="map"
	default = {
		
		adapter_name=""
		auth_name=""
		chap_secret=""
		send_target=""
	}
}

resource "vsphere_host" "h1"{
	name =""
	datacenter_id = "${data.vsphere_datacenter.dc.id}"
	host_config = "${var.config1}"
	iscsi_config = "${var.iscsi_config}"
}