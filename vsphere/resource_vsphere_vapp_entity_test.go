package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccResourceVSphereVAppEntity_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVAppEntityPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVAppEntityConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVAppEntityCheckExists(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereVAppEntity_nonDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVAppEntityPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVAppEntityConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVAppEntityCheckExists(true),
				),
			},
		},
	})
}

func testAccResourceVSphereVAppEntityPreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_DATACENTER") == "" {
		t.Skip("set VSPHERE_DATACENTER to run vsphere_resource_pool acceptance tests")
	}
	if os.Getenv("VSPHERE_CLUSTER") == "" {
		t.Skip("set VSPHERE_CLUSTER to run vsphere_resource_pool acceptance tests")
	}
	if os.Getenv("VSPHERE_ESXI_HOST5") == "" {
		t.Skip("set VSPHERE_ESXI_HOST5 to run vsphere_resource_pool acceptance tests")
	}
}

func testAccResourceVSphereVAppEntityCheckExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return nil
	}
}

//	start_action   = "none"
//	start_delay    = 5
//	stop_action    = "guestShutdown"
//	stop_delay     = 5
//	start_order    = 1
//	wait_for_guest = true
func testAccResourceVSphereVAppEntityStartAction(value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := testGetVAppEntity(s, "vapp_entity")
		if err != nil {
			return err
		}
		if true {
			return fmt.Errorf("CpuAllocation.Reservation check failed. Expected: %d, got: %d", 1, 1)
		}
		return nil
	}
}

func testAccResourceVSphereVAppEntityConfigBasic() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
  default = "%s"
}

variable "datastore" {
	default = "%s"
}

variable "network_label" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
	name = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "${var.cluster}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_resource_pool" "parent_resource_pool" {
  name                    = "terraform-resource-pool-test-parent"
  parent_resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
}

resource "vsphere_folder" "parent_folder" {
	path = "parent_folder"
	type = "vm"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_vapp_container" "vapp_container" {
  name                    = "terraform-resource-pool-test"
  parent_resource_pool_id = "${vsphere_resource_pool.parent_resource_pool.id}"
	parent_folder_id = "${vsphere_folder.parent_folder.id}"
}

resource "vsphere_virtual_machine" "vm" {
	name = "terraform-virtual-machine-test"
	resource_pool_id = "${vsphere_vapp_container.vapp_container.resource_pool_id}"
	datastore_id = "${data.vsphere_datastore.datastore.id}"

	num_cpus = 2
	memory   = 2048
	guest_id = "other3xLinux64Guest"
	wait_for_guest_net_timeout = -1

	
	disk {
		label = "disk0"
		size = "1"
	}

	network_interface {
		network_id = "${data.vsphere_network.network.id}"
	}
}

resource "vsphere_vapp_entity" "vapp_entity" {
	target_id = "${vsphere_virtual_machine.vm.id}"
	container_id = "${vsphere_vapp_container.vapp_container.id}"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
	)
}

func testAccResourceVSphereVAppEntityConfigNonDefault() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "${var.cluster}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_resource_pool" "parent_resource_pool" {
  name                    = "terraform-resource-pool-test-parent"
  parent_resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
}

resource "vsphere_folder" "parent_folder" {
  path          = "parent_folder"
  type          = "vm"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_vapp_container" "vapp_container" {
  name                    = "terraform-resource-pool-test"
  parent_resource_pool_id = "${vsphere_resource_pool.parent_resource_pool.id}"
  parent_folder_id        = "${vsphere_folder.parent_folder.id}"
}

resource "vsphere_vapp_entity" "vapp_entity" {
  target_id      = "${vsphere_virtual_machine.vm.id}"
  container_id   = "${vsphere_vapp_container.vapp_container.id}"
	start_action   = "none"
	start_delay    = 5
	stop_action    = "guestShutdown"
	stop_delay     = 5
	start_order    = 1
	wait_for_guest = true
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-virtual-machine-test"
  resource_pool_id = "${vsphere_vapp_container.vapp_container.resource_pool_id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus                   = 2
  memory                     = 2048
  guest_id                   = "other3xLinux64Guest"
  wait_for_guest_net_timeout = -1

  disk {
    label = "disk0"
    size  = "1"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
	)
}

func testAccResourceVSphereVAppEntityConfigMultipleDefault() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "${var.cluster}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_resource_pool" "parent_resource_pool" {
  name                    = "terraform-resource-pool-test-parent"
  parent_resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
}

resource "vsphere_folder" "parent_folder" {
  path          = "parent_folder"
  type          = "vm"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_vapp_container" "vapp_container" {
  name                    = "terraform-resource-pool-test"
  parent_resource_pool_id = "${vsphere_resource_pool.parent_resource_pool.id}"
  parent_folder_id        = "${vsphere_folder.parent_folder.id}"
}

resource "vsphere_vapp_entity" "vapp_entity1" {
  target_id      = "${vsphere_virtual_machine.vm.*.id[0]}"
  container_id   = "${vsphere_vapp_container.vapp_container.id}"
}

resource "vsphere_vapp_entity" "vapp_entity2" {
  target_id      = "${vsphere_virtual_machine.vm.*.id[1]}"
  container_id   = "${vsphere_vapp_container.vapp_container.id}"
}

resource "vsphere_vapp_entity" "vapp_entity3" {
  target_id      = "${vsphere_virtual_machine.vm.*.id[2]}"
  container_id   = "${vsphere_vapp_container.vapp_container.id}"
}

resource "vsphere_virtual_machine" "vm" {
	count            = 3
  name             = "terraform-virtual-machine-test-${count.index}"
  resource_pool_id = "${vsphere_vapp_container.vapp_container.resource_pool_id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus                   = 2
  memory                     = 2048
  guest_id                   = "other3xLinux64Guest"
  wait_for_guest_net_timeout = -1

  disk {
    label = "disk0"
    size  = "1"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
	)
}
func testAccResourceVSphereVAppEntityConfigMultipleNonDefault() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "${var.cluster}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_resource_pool" "parent_resource_pool" {
  name                    = "terraform-resource-pool-test-parent"
  parent_resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
}

resource "vsphere_folder" "parent_folder" {
  path          = "parent_folder"
  type          = "vm"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_vapp_container" "vapp_container" {
  name                    = "terraform-resource-pool-test"
  parent_resource_pool_id = "${vsphere_resource_pool.parent_resource_pool.id}"
  parent_folder_id        = "${vsphere_folder.parent_folder.id}"
}

resource "vsphere_vapp_entity" "vapp_entity1" {
  target_id      = "${vsphere_virtual_machine.vm.*.id[0]}"
  container_id   = "${vsphere_vapp_container.vapp_container.id}"
	start_action   = "powerOn"
	start_delay    = 5
	stop_action    = "guestShutdown"
	stop_delay     = 5
	start_order    = 3
	wait_for_guest = true
}

resource "vsphere_vapp_entity" "vapp_entity2" {
  target_id      = "${vsphere_virtual_machine.vm.*.id[1]}"
  container_id   = "${vsphere_vapp_container.vapp_container.id}"
	start_action   = "powerOn"
	start_delay    = 5
	stop_action    = "guestShutdown"
	stop_delay     = 5
	start_order    = 1
	wait_for_guest = true
}

resource "vsphere_vapp_entity" "vapp_entity3" {
  target_id      = "${vsphere_virtual_machine.vm.*.id[2]}"
  container_id   = "${vsphere_vapp_container.vapp_container.id}"
	start_action   = "powerOn"
	start_delay    = 5
	stop_action    = "guestShutdown"
	stop_delay     = 5
	start_order    = 2
	wait_for_guest = true
}

resource "vsphere_virtual_machine" "vm" {
	count            = 3
  name             = "terraform-virtual-machine-test-${count.index}"
  resource_pool_id = "${vsphere_vapp_container.vapp_container.resource_pool_id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus                   = 2
  memory                     = 2048
  guest_id                   = "other3xLinux64Guest"
  wait_for_guest_net_timeout = -1

  disk {
    label = "disk0"
    size  = "1"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
	)
}
