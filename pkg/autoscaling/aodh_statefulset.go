/*
Copyright 2022.

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

// Package autoscaling provides functionality for managing OpenStack telemetry autoscaling components
package autoscaling

import (
	"fmt"
	"strings"

	"github.com/openstack-k8s-operators/lib-common/modules/common/annotations"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/service"
	"github.com/openstack-k8s-operators/lib-common/modules/common/tls"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	memcachedv1 "github.com/openstack-k8s-operators/infra-operator/apis/memcached/v1beta1"
	topologyv1 "github.com/openstack-k8s-operators/infra-operator/apis/topology/v1beta1"
	telemetryv1 "github.com/openstack-k8s-operators/telemetry-operator/api/v1beta1"
)

// AodhStatefulSet func
func AodhStatefulSet(
	instance *telemetryv1.Autoscaling,
	configHash string,
	labels map[string]string,
	topology *topologyv1.Topology,
	memcached *memcachedv1.Memcached,
) (*appsv1.StatefulSet, error) {
	livenessProbe := &corev1.Probe{
		// TODO might need tuning
		TimeoutSeconds:      30,
		PeriodSeconds:       30,
		InitialDelaySeconds: 5,
	}
	readinessProbe := &corev1.Probe{
		// TODO might need tuning
		TimeoutSeconds:      30,
		PeriodSeconds:       30,
		InitialDelaySeconds: 5,
	}

	args := []string{"-c"}
	// Adjust the args to execute the start command directly
	// instead of running kolla_start
	argsApi := append(args, "/usr/sbin/httpd -DFOREGROUND -E /dev/stdout")
	argsEvaluator := append(args, "/usr/bin/aodh-evaluator --logfile /dev/stdout")
	argsListener := append(args, "/usr/bin/aodh-listener --logfile /dev/stdout")
	argsNotifier := append(args, "/usr/bin/aodh-notifier --logfile /dev/stdout")

	livenessProbe.HTTPGet = &corev1.HTTPGetAction{
		Path: "/",
		Port: intstr.IntOrString{Type: intstr.Int, IntVal: int32(AodhAPIPort)},
	}
	readinessProbe.HTTPGet = &corev1.HTTPGetAction{
		Path: "/",
		Port: intstr.IntOrString{Type: intstr.Int, IntVal: int32(AodhAPIPort)},
	}

	if instance.Spec.Aodh.TLS.API.Enabled(service.EndpointPublic) {
		livenessProbe.HTTPGet.Scheme = corev1.URISchemeHTTPS
		readinessProbe.HTTPGet.Scheme = corev1.URISchemeHTTPS
	}

	// create Volume and VolumeMounts
	volumes := getVolumes()
	apiVolumeMounts := getVolumeMounts("aodh-api")
	evaluatorVolumeMounts := getVolumeMounts("aodh-evaluator")
	notifierVolumeMounts := getVolumeMounts("aodh-notifier")
	listenerVolumeMounts := getVolumeMounts("aodh-listener")

	// add openstack CA cert if defined
	if instance.Spec.Aodh.TLS.CaBundleSecretName != "" {
		volumes = append(volumes, instance.Spec.Aodh.TLS.CreateVolume())
		apiVolumeMounts = append(apiVolumeMounts, instance.Spec.Aodh.TLS.CreateVolumeMounts(nil)...)
		evaluatorVolumeMounts = append(evaluatorVolumeMounts, instance.Spec.Aodh.TLS.CreateVolumeMounts(nil)...)
		notifierVolumeMounts = append(notifierVolumeMounts, instance.Spec.Aodh.TLS.CreateVolumeMounts(nil)...)
		listenerVolumeMounts = append(listenerVolumeMounts, instance.Spec.Aodh.TLS.CreateVolumeMounts(nil)...)
	}

	// add prometheus CA cert if defined
	if instance.Spec.PrometheusTLSCaCertSecret != nil {
		volumes = append(volumes, getCustomPrometheusCaVolume(instance.Spec.PrometheusTLSCaCertSecret.Name))
		evaluatorVolumeMounts = append(evaluatorVolumeMounts, getCustomPrometheusCaVolumeMount(instance.Spec.PrometheusTLSCaCertSecret.Key))
	}

	// add MTLS cert if defined
	if memcached.GetMemcachedMTLSSecret() != "" {
		volumes = append(volumes, memcached.CreateMTLSVolume())
		// NOTE: This would need the same as below
		apiVolumeMounts = append(apiVolumeMounts, memcached.CreateMTLSVolumeMounts(nil, nil)...)
	}

	for _, endpt := range []service.Endpoint{service.EndpointInternal, service.EndpointPublic} {
		if instance.Spec.Aodh.TLS.API.Enabled(endpt) {
			var tlsEndptCfg tls.GenericService
			switch endpt {
			case service.EndpointPublic:
				tlsEndptCfg = instance.Spec.Aodh.TLS.API.Public
			case service.EndpointInternal:
				tlsEndptCfg = instance.Spec.Aodh.TLS.API.Internal
			}

			svc, err := tlsEndptCfg.ToService()
			if err != nil {
				return nil, err
			}
			volumes = append(volumes, svc.CreateVolume(endpt.String()))
			// Modify the MountPath of TLS related files to mount
			// them to their final location instead of relying
			// on kolla
			certs := svc.CreateVolumeMounts(endpt.String())
			for _, cert := range certs {
				// NOTE: This could be done in libcommon
				cert.MountPath = strings.Replace(cert.MountPath, "/var/lib/config-data/tls/", "/etc/pki/tls/", 1)
				apiVolumeMounts = append(apiVolumeMounts, cert)
			}
		}
	}

	envVarsAodh := map[string]env.Setter{}
	envVarsAodh["KOLLA_CONFIG_STRATEGY"] = env.SetValue("COPY_ALWAYS")
	envVarsAodh["CONFIG_HASH"] = env.SetValue(configHash)

	var replicas int32 = 1

	apiContainer := corev1.Container{
		ImagePullPolicy: corev1.PullAlways,
		Command: []string{
			"/bin/bash",
		},
		Args:         argsApi,
		Image:        instance.Spec.Aodh.APIImage,
		Name:         "aodh-api",
		Env:          env.MergeEnvs([]corev1.EnvVar{}, envVarsAodh),
		VolumeMounts: apiVolumeMounts,
		// Not using kolla alows us to disallow privilege escalation
		// and drop all capabilities (there may be reasons why
		// this wouldn't be possible for some other services)
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{
					"ALL",
				},
			},
		},
	}

	evaluatorContainer := corev1.Container{
		ImagePullPolicy: corev1.PullAlways,
		Command: []string{
			"/bin/bash",
		},
		Args:         argsEvaluator,
		Image:        instance.Spec.Aodh.EvaluatorImage,
		Name:         "aodh-evaluator",
		Env:          env.MergeEnvs([]corev1.EnvVar{}, envVarsAodh),
		VolumeMounts: evaluatorVolumeMounts,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{
					"ALL",
				},
			},
		},
	}

	notifierContainer := corev1.Container{
		ImagePullPolicy: corev1.PullAlways,
		Command: []string{
			"/bin/bash",
		},
		Args:         argsNotifier,
		Image:        instance.Spec.Aodh.NotifierImage,
		Name:         "aodh-notifier",
		Env:          env.MergeEnvs([]corev1.EnvVar{}, envVarsAodh),
		VolumeMounts: notifierVolumeMounts,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{
					"ALL",
				},
			},
		},
	}

	listenerContainer := corev1.Container{
		ImagePullPolicy: corev1.PullAlways,
		Command: []string{
			"/bin/bash",
		},
		Args:         argsListener,
		Image:        instance.Spec.Aodh.ListenerImage,
		Name:         "aodh-listener",
		Env:          env.MergeEnvs([]corev1.EnvVar{}, envVarsAodh),
		VolumeMounts: listenerVolumeMounts,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{
					"ALL",
				},
			},
		},
	}

	pod := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: instance.RbacResourceName(),
			Containers: []corev1.Container{
				apiContainer,
				evaluatorContainer,
				notifierContainer,
				listenerContainer,
			},
			// I don't think there is a need to run anything with
			// Aodh's UUID. Removing the RunAsUser, but having
			// RunAsNonRoot set to true will allow
			// OCP to decide what UUID to use in order to comply
			// with security constraints (there is an interval
			// in which some constraints require the UUID to be).

			// In this configuration OCP will also fill in
			// FSGroup with the same ID. So each mounted volume
			// will be owned by the same group as is being used
			// to run Aodh. Permissions on the volumes will be
			// OR'd with 0660 unless the volume is read only, in
			// which case they seem to be OR'd with 0440. This
			// means we don't need kolla to set the ownership
			// and permissions of files.
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot:       ptr.To(true),
				// We still need the aodh group to access
				// some of the files installed with the RPM
				// that are owned by aodh:aodh. Security
				// constraints don't seem to mind this.
				SupplementalGroups: []int64{AodhUserID},
				// Setting seccompprofile is required by
				// some of the security constraints
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
		},
	}

	if instance.Spec.Aodh.NodeSelector != nil {
		pod.Spec.NodeSelector = *instance.Spec.Aodh.NodeSelector
	}
	if topology != nil {
		topology.ApplyTo(&pod)
	}

	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Replicas:            &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: pod,
		},
	}

	statefulset.Spec.Template.Spec.Volumes = volumes

	// networks to attach to
	nwAnnotation, err := annotations.GetNADAnnotation(instance.Namespace, instance.Spec.Aodh.NetworkAttachmentDefinitions)
	if err != nil {
		return nil, fmt.Errorf("failed create network annotation from %s: %w",
			instance.Spec.Aodh.NetworkAttachmentDefinitions, err)
	}
	statefulset.Spec.Template.Annotations = util.MergeStringMaps(statefulset.Spec.Template.Annotations, nwAnnotation)

	return statefulset, nil
}
