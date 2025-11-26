/*

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

// Package telemetry provides constants and utilities for OpenStack telemetry service management
package telemetry

import (
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
)

const (
	// ServiceName -
	ServiceName = "telemetry"
	// ServiceType -
	ServiceType = "telemetry"
	// DpdkServiceName -
	DpdkServiceName = "configure-ovs-dpdk"
	// Telemetry is the global ServiceType that refers to all the components deployed
	// by the telemetry operator
	Telemetry storage.PropagationType = "Telemetry"
)

var TelemetryVolumePropagation = []storage.PropagationType{Telemetry}
