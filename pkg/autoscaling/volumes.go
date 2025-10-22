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

package autoscaling

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	scriptVolume = "aodh-scripts"
	configVolume = "aodh-config-data"
	logVolume    = "logs"
)

var (
	// scriptMode is the default permissions mode for Scripts volume
	scriptMode int32 = 0740
	// configMode is the 640 permissions mode
	configMode int32 = 0640
)

// getVolumes - service volumes
func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "scripts",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &scriptMode,
					SecretName:  scriptVolume,
				},
			},
		}, {
			Name: "config-data",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &configMode,
					SecretName:  configVolume,
				},
			},
		}, {
			Name: "run",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}

// getVolumeMounts - general VolumeMounts
func getVolumeMounts(serviceName string) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "scripts",
			MountPath: "/var/lib/openstack/bin",
			ReadOnly:  true,
		},
		// We're not using kolla to copy files around. So we
		// need to specify the correct MountPath, SubPath
		// for each file.
		{
			Name:      "config-data",
			MountPath: "/etc/aodh/aodh.conf",
			SubPath:   "aodh.conf",
			ReadOnly:  true,
		},
		{
			Name:      "config-data",
			MountPath: "/etc/aodh/aodh.conf.d/01-aodh-custom.conf",
			SubPath:   "custom.conf",
			ReadOnly:  true,
		},
		{
			Name:      "config-data",
			MountPath: "/etc/httpd/conf.d/00wsgi-aodh.conf",
			SubPath:   "wsgi-aodh.conf",
			ReadOnly:  true,
		},
		{
			Name:      "config-data",
			MountPath: "/etc/httpd/conf/httpd.conf",
			SubPath:   "httpd.conf",
			ReadOnly:  true,
		},
		{
			Name:      "config-data",
			MountPath: "/etc/httpd/conf.d/ssl.conf",
			SubPath:   "ssl.conf",
			ReadOnly:  true,
		},
		{
			Name:      "config-data",
			MountPath: "/etc/openstack/prometheus.yaml",
			SubPath:   "prometheus.yaml",
			ReadOnly:  true,
		},
		// Seems like httpd needs an accessible directory to store
		// its pid file(s). This seems like a working solution
		{
			Name:      "run",
			MountPath: "/etc/httpd/run",
		},
		{
			Name:      "config-data",
			MountPath: "/etc/my.cnf",
			SubPath:   "my.cnf",
			ReadOnly:  true,
		},
	}
}

// getCustomPrometheusCaVolume - Volume for CA certificate of user deployed Prometheus
func getCustomPrometheusCaVolume(secretName string) corev1.Volume {
	return corev1.Volume{
		Name: "custom-prometheus-ca",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}
}

// getCustomPrometheusCaVolumeMount - VolumeMount for CA certificate of user deployed Prometheus
func getCustomPrometheusCaVolumeMount(fileName string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      "custom-prometheus-ca",
		MountPath: CustomPrometheusCaCertFolderPath + fileName,
		SubPath:   fileName,
		ReadOnly:  true,
	}
}
