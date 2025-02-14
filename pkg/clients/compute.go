/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clients

import (
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/attachinterfaces"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/utils/openstack/compute/v2/flavors"

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/metrics"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/scope"
)

/*
NovaMinimumMicroversion is the minimum Nova microversion supported by CAPO
2.53 corresponds to OpenStack Pike

For the canonical description of Nova microversions, see
https://docs.openstack.org/nova/latest/reference/api-microversion-history.html

CAPO uses server tags, which were added in microversion 2.52.
*/
const NovaMinimumMicroversion = "2.53"

// ServerExt is the base gophercloud Server with extensions used by InstanceStatus.
type ServerExt struct {
	servers.Server
	availabilityzones.ServerAvailabilityZoneExt
}

type ComputeClient interface {
	ListAvailabilityZones() ([]availabilityzones.AvailabilityZone, error)

	GetFlavorIDFromName(flavor string) (string, error)
	CreateServer(createOpts servers.CreateOptsBuilder) (*ServerExt, error)
	DeleteServer(serverID string) error
	GetServer(serverID string) (*ServerExt, error)
	ListServers(listOpts servers.ListOptsBuilder) ([]ServerExt, error)

	ListAttachedInterfaces(serverID string) ([]attachinterfaces.Interface, error)
	DeleteAttachedInterface(serverID, portID string) error
}

type computeClient struct{ client *gophercloud.ServiceClient }

// NewComputeClient returns a new compute client.
func NewComputeClient(scope *scope.Scope) (ComputeClient, error) {
	compute, err := openstack.NewComputeV2(scope.ProviderClient, gophercloud.EndpointOpts{
		Region: scope.ProviderClientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service client: %v", err)
	}
	compute.Microversion = NovaMinimumMicroversion

	return &computeClient{compute}, nil
}

func (c computeClient) ListAvailabilityZones() ([]availabilityzones.AvailabilityZone, error) {
	mc := metrics.NewMetricPrometheusContext("availability_zone", "list")
	allPages, err := availabilityzones.List(c.client).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return availabilityzones.ExtractAvailabilityZones(allPages)
}

func (c computeClient) GetFlavorIDFromName(flavor string) (string, error) {
	mc := metrics.NewMetricPrometheusContext("flavor", "get")
	flavorID, err := flavors.IDFromName(c.client, flavor)
	return flavorID, mc.ObserveRequest(err)
}

func (c computeClient) CreateServer(createOpts servers.CreateOptsBuilder) (*ServerExt, error) {
	var server ServerExt
	mc := metrics.NewMetricPrometheusContext("server", "create")
	err := servers.Create(c.client, createOpts).ExtractInto(&server)
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return &server, nil
}

func (c computeClient) DeleteServer(serverID string) error {
	mc := metrics.NewMetricPrometheusContext("server", "delete")
	err := servers.Delete(c.client, serverID).ExtractErr()
	return mc.ObserveRequestIgnoreNotFound(err)
}

func (c computeClient) GetServer(serverID string) (*ServerExt, error) {
	var server ServerExt
	mc := metrics.NewMetricPrometheusContext("server", "get")
	err := servers.Get(c.client, serverID).ExtractInto(&server)
	if mc.ObserveRequestIgnoreNotFound(err) != nil {
		return nil, err
	}
	return &server, nil
}

func (c computeClient) ListServers(listOpts servers.ListOptsBuilder) ([]ServerExt, error) {
	var serverList []ServerExt
	mc := metrics.NewMetricPrometheusContext("server", "list")
	allPages, err := servers.List(c.client, listOpts).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	err = servers.ExtractServersInto(allPages, &serverList)
	return serverList, err
}

func (c computeClient) ListAttachedInterfaces(serverID string) ([]attachinterfaces.Interface, error) {
	mc := metrics.NewMetricPrometheusContext("server_os_interface", "list")
	interfaces, err := attachinterfaces.List(c.client, serverID).AllPages()
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}
	return attachinterfaces.ExtractInterfaces(interfaces)
}

func (c computeClient) DeleteAttachedInterface(serverID, portID string) error {
	mc := metrics.NewMetricPrometheusContext("server_os_interface", "delete")
	err := attachinterfaces.Delete(c.client, serverID, portID).ExtractErr()
	return mc.ObserveRequestIgnoreNotFoundorConflict(err)
}

type computeErrorClient struct{ error }

// NewComputeErrorClient returns a ComputeClient in which every method returns the given error.
func NewComputeErrorClient(e error) ComputeClient {
	return computeErrorClient{e}
}

func (e computeErrorClient) ListAvailabilityZones() ([]availabilityzones.AvailabilityZone, error) {
	return nil, e.error
}

func (e computeErrorClient) GetFlavorIDFromName(flavor string) (string, error) {
	return "", e.error
}

func (e computeErrorClient) CreateServer(createOpts servers.CreateOptsBuilder) (*ServerExt, error) {
	return nil, e.error
}

func (e computeErrorClient) DeleteServer(serverID string) error {
	return e.error
}

func (e computeErrorClient) GetServer(serverID string) (*ServerExt, error) {
	return nil, e.error
}

func (e computeErrorClient) ListServers(listOpts servers.ListOptsBuilder) ([]ServerExt, error) {
	return nil, e.error
}

func (e computeErrorClient) ListAttachedInterfaces(serverID string) ([]attachinterfaces.Interface, error) {
	return nil, e.error
}

func (e computeErrorClient) DeleteAttachedInterface(serverID, portID string) error {
	return e.error
}
