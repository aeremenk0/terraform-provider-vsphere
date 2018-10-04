data "vsphere_datacenter" "dc"{
	name = "BAUM"
}

variable "config1" {
	type="map"
	default = {
		dns="asdf"
		root_password="jP1!cH@s"
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
	name ="icpcdc200847.svr.us.jpmchase.net"
	datacenter_id = "${data.vsphere_datacenter.dc.id}"
	host_config = "${var.config1}"
	iscsi_config = "${var.iscsi_config}"
}