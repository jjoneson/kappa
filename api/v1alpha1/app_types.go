/*
Copyright 2021.

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

package v1alpha1

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AppSpec defines the desired state of App
type AppSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//+kubebuilder:validation:Optional
	// Application Version
	Version string `json:"version,omitempty"`

	//+kubebuilder:validation:Required
	// Image of application
	Image string `json:"image"`

	//+kubebuilder:validation:Optional
	// Image Digest
	ImageDigest string `json:"imageDigest,omitempty"`

	//+kubebuilder:validation:Optional
	// Image Pull Secrets
	ImagePullSecrets string `json:"imagePullSecrets,omitempty"`

	// Instances/Replicas
	//+kubebuilder:validation:Optional
	// +kubebuilder:default:=1
	Instances *int32 `json:"instances,omitempty"`

	// Memory Request/Limit, defaults to 256Mi
	//+kubebuilder:validation:Optional
	// +kubebuilder:default:="256Mi"
	Memory string `json:"memory,omitempty"`

	// Cpu Request/Limit, defaults to 200m
	//+kubebuilder:validation:Optional
	// +kubebuilder:default:="200m"
	Cpu string `json:"cpu,omitempty"`

	// Port, defaults to 8080
	//+kubebuilder:validation:Optional
	// +kubebuilder:default:=8080
	Port *int32 `json:"port,omitempty"`

	// Public Hostname, defaults to app name
	//+kubebuilder:validation:Optional
	Hostname string `json:"hostname,omitempty"`

	// Disable Istio MTLS, defaults to false
	//+kubebuilder:validation:Optional
	DisableMtls *bool `json:"disableMtls,omitempty"`

	//+kubebuilder:validation:Optional
	// Expose route through ingress gateway, defaults to true
	Public *bool `json:"public,omitempty"`

	//+kubebuilder:validation:Optional
	// Node Selector
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	//+kubebuilder:validation:Optional
	// Disable istio sidecar, defaults to false
	DisableSidecar *bool `json:"disableSidecar,omitempty"`

	//+kubebuilder:validation:Optional
	// Environment Variables
	Env []v1.EnvVar `json:"env,omitempty"`

	//+kubebuilder:validation:Optional
	// Secrets to mount as environment variables
	Secrets []string `json:"secrets,omitempty"`

	//+kubebuilder:validation:Optional
	// Annotations to add to all resources
	Annotations map[string]string `json:"annotations,omitempty"`

	//+kubebuilder:validation:Optional
	// Labels to add to all resources
	Labels map[string]string `json:"labels,omitempty"`

	//+kubebuilder:validation:Optional
	// Config to store in configmap and mount as files
	Config map[string]string `json:"config,omitempty"`

	//+kubebuilder:validation:Optional
	// +kubebuilder:default:="tcp"
	// Health Check type, defaults to "tcp"
	HealthCheckType string `json:"healthCheckType"`

	//+kubebuilder:validation:Optional
	// Endpoint for health check if set to Http
	HealthCheckEndpoint string `json:"healthCheckEndpoint"`
}

// AppStatus defines the observed state of App
type AppStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	appsv1.DeploymentStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// App is the Schema for the apps API
type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppSpec   `json:"spec,omitempty"`
	Status AppStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AppList contains a list of App
type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []App `json:"items"`
}

func init() {
	SchemeBuilder.Register(&App{}, &AppList{})
}
