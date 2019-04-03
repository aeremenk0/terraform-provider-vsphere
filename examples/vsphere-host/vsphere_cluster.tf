provider "vsphere" {
	user="administrator@vsphere.local"
	password="VMware1!"
	vsphere_server="192.168.75.100"
	allow_unverified_ssl=true
}


data "vsphere_datacenter" "dc"{
	name = "Datacenter"
}

variable "advanced" {
	type="map"
	default = {
		UserVars_HostClientWelcomeMessage = "This welcome message has been changed through the advanced options"
	}
}

variable "config1" {
	type="map"
	default = {
		default_ip="192.168.75.135"
		default_username="root"
		default_password="VMware1!"
		root_password="VMware1!"
		enable_ssh=true
		connected=true
		maintenance_mode=true
	}
}

variable "iscsi_config2" {
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


resource "vsphere_host" "h2"{
	name = "192.168.75.134"
	datacenter_id = "${data.vsphere_datacenter.dc.id}"
	host_config = "${var.config1}"
	iscsi_config = "${var.iscsi_config2}"
	advanced_options = "${var.advanced}"
	
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name            = "terraform-compute-cluster-test"
  datacenter_id   = "${data.vsphere_datacenter.dc.id}"
  host_system_ids = ["${vsphere_host.h2.id}"]
  force_evacuate_on_destroy = true
  drs_enabled          = false

  ha_enabled = false
}

resource "vsphere_compute_cluster_host_group" "cluster_host_group" {
  name                = "terraform-test-cluster-host-group"
  compute_cluster_id  = "${vsphere_compute_cluster.compute_cluster.id}"
  host_system_ids     = ["${vsphere_host.h2.id}"]
}

resource "vsphere_host_virtual_switch" "switch" {
  name           = "vSwitch0"
  host_system_id = "${vsphere_host.h2.id}"
  
  network_adapters = ["vmnic0"]
  mtu = 1500

  active_nics  = ["vmnic0"]
  standby_nics = []
}

resource "vsphere_host_port_group" "pg" {
  name                = "PGTerraformTest"
  host_system_id      = "${vsphere_host.h2.id}"
  virtual_switch_name = "${vsphere_host_virtual_switch.switch.name}"
  
  vlan_id = 1
  
}
