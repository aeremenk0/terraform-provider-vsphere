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
		dns="asdf"
		root_password="VMware1!"
		ntp_server="asdf"
		enable_ssh=true
		connected=true
	}
}

variable "config2" {
	type="map"
	default = {
		dns="asdf"
		root_password="VMware1!"
		ntp_server="asdf"
		enable_ssh=true
		connected=true
	}
}

variable "iscsi_config1" {
	type="map"
	default = {
		adapter_name="test"
		auth_name="test"
		chap_secret="test"
		send_target="test"
	}
}

variable "iscsi_config2" {
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
	iscsi_config = "${var.iscsi_config1}"
	
}

resource "vsphere_host" "h2"{
	name = "192.168.75.134"
	datacenter_id = "${data.vsphere_datacenter.dc.id}"
	host_config = "${var.config2}"
	iscsi_config = "${var.iscsi_config2}"
	
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name            = "terraform-compute-cluster-test"
  datacenter_id   = "${data.vsphere_datacenter.dc.id}"
  host_system_ids = ["${vsphere_host.h1.id}","${vsphere_host.h2.id}"]

  drs_enabled          = false

  ha_enabled = false
}

resource "vsphere_compute_cluster_host_group" "cluster_host_group" {
  name                = "terraform-test-cluster-host-group"
  compute_cluster_id  = "${vsphere_compute_cluster.compute_cluster.id}"
  host_system_ids     = ["${vsphere_host.h1.id}","${vsphere_host.h2.id}"]
}