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

// Package ceilometer provides functionality for managing OpenStack Ceilometer telemetry components
package ceilometer

import (
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	"github.com/openstack-k8s-operators/telemetry-operator/internal/telemetry"
)

const (
	// ServiceName -
	ServiceName = "ceilometer"
	// ComputeServiceName -
	ComputeServiceName = "ceilometer-compute"
	// IpmiServiceName -
	IpmiServiceName = "ceilometer-ipmi"
	// ServiceType -
	ServiceType = "Ceilometer"

	// CeilometerPrometheusPort -
	CeilometerPrometheusPort int = 3000

	// KollaConfigCentral -
	KollaConfigCentral = "/var/lib/config-data/merged/config-central.json"

	// KollaConfigNotification -
	KollaConfigNotification = "/var/lib/config-data/merged/config-notification.json"

	// CeilometerUserID -
	CeilometerUserID = 42405

	// Ceilometer is the global ServiceType that refers to all the components deployed
	// by the Ceilometer controller
	Ceilometer storage.PropagationType = "Ceilometer"
)

// CeilometerPropagation is the definition of the Ceilometer propagation group
// It allows the Ceilometer pod to mount volumes destined to Ceilometer related
// ServiceTypes
var CeilometerPropagation = []storage.PropagationType{telemetry.Telemetry, Ceilometer}
