package controllers

import (
	"context"
	"fmt"
	kappv1alpha1 "github.com/jjoneson/kappa/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *AppReconciler) reconcileDeployment(ctx context.Context, req ctrl.Request, app *kappv1alpha1.App) (ctrl.Result, error) {
	found := &appsv1.Deployment{}
	desired := r.deployment(app)
	err := controllerutil.SetControllerReference(app, desired, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = r.Create(ctx, desired); err != nil {
				return ctrl.Result{}, err
			}
			r.Log.Info("Created new deployment", "Name", app.Name, "Namespace", app.Namespace)
			return ctrl.Result{Requeue: true}, nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if !reflect.DeepEqual(found.Status, app.Status.DeploymentStatus) {
		found.Status.DeepCopyInto(&app.Status.DeploymentStatus)
		if err := r.Status().Update(ctx, app); err != nil {
			return ctrl.Result{}, err
		}
		r.Log.Info("Updating Status deployment", "Name", app.Name, "Namespace", app.Namespace)
	}

	if !r.deploymentEquality(desired, found) {
		desired.DeepCopyInto(found)
		r.Log.Info("Updating deployment", "Name", app.Name, "Namespace", app.Namespace)
		if err := r.Update(ctx, found); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *AppReconciler) deployment(app *kappv1alpha1.App) *appsv1.Deployment {

	maxUnavailable := 1
	if app.Spec.Instances != nil && *app.Spec.Instances == 1 {
		maxUnavailable = 0
	}
	var podAnnotations interface{}
	if app.Spec.Annotations != nil {
		podAnnotations = runtime.DeepCopyJSONValue(app.Spec.Annotations)
	} else {
		podAnnotations = make(map[string]string)
	}

	podAnnotations.(map[string]string)["sidecar.istio.io/rewriteAppHTTPProbers"] = "true"
	podAnnotations.(map[string]string)["seccomp.security.alpha.kubernetes.io/pod"] = "runtime/default"
	podAnnotations.(map[string]string)["cluster-autoscaler.kubernetes.io/safe-to-evict"] = "true"

	if app.Spec.DisableSidecar != nil && *app.Spec.DisableSidecar == false {
		podAnnotations.(map[string]string)["sidecar.istio.io/inject"] = "false"
	}
	var matchExpressions []metav1.LabelSelectorRequirement
	if app.Spec.NodeSelector != nil {
		for k, v := range app.Spec.NodeSelector {
			matchExpressions = append(matchExpressions, metav1.LabelSelectorRequirement{
				Key:      k,
				Operator: "In",
				Values:   []string{v},
			})
		}
	}

	var envFrom []corev1.EnvFromSource
	if app.Spec.Secrets != nil {
		for _, secret := range app.Spec.Secrets {
			envFrom = append(envFrom, corev1.EnvFromSource{SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secret,
				},
			}})
		}
	}

	labels := make(map[string]string)
	for k, v := range app.Labels {
		labels[k] = v
	}
	labels["app"] = app.Name

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        app.Name,
			Namespace:   app.Namespace,
			Labels:      labels,
			Annotations: app.Spec.Annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: app.Spec.Instances,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: int32(maxUnavailable)},
					MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": app.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        app.Name,
					Namespace:   app.Namespace,
					Labels:      labels,
					Annotations: podAnnotations.(map[string]string),
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: pointer.Int64Ptr(1000),
					},
					ServiceAccountName: app.Name,
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: matchExpressions,
									},
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
					NodeSelector: app.Spec.NodeSelector,
					Containers: []corev1.Container{
						{
							Name:            app.Name,
							Image:           imageName(app),
							Env:             app.Spec.Env,
							EnvFrom:         envFrom,
							ImagePullPolicy: corev1.PullAlways,
							ReadinessProbe:  probe(app, 10, 12),
							LivenessProbe:   probe(app, 120, 1),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: *app.Spec.Port,
									Protocol:      "TCP",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(app.Spec.Cpu),
									corev1.ResourceMemory: resource.MustParse(app.Spec.Memory),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(app.Spec.Cpu),
									corev1.ResourceMemory: resource.MustParse(app.Spec.Memory),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										"ALL",
									},
								},
								Privileged:               pointer.BoolPtr(false),
								RunAsUser:                pointer.Int64Ptr(1000),
								RunAsGroup:               pointer.Int64Ptr(1000),
								RunAsNonRoot:             pointer.BoolPtr(true),
								ReadOnlyRootFilesystem:   pointer.BoolPtr(false),
								AllowPrivilegeEscalation: pointer.BoolPtr(false),
							},
						},
					},
				},
			},
		},
	}
}

func imageName(app *kappv1alpha1.App) string {
	image := app.Spec.Image
	if app.Spec.Version != "" {
		image = fmt.Sprintf("%s:%s", image, app.Spec.Version)
	} else if app.Spec.ImageDigest != "" {
		image = fmt.Sprintf("%s@%s", image, app.Spec.ImageDigest)
	} else {
		image = fmt.Sprintf("%s:latest", image)
	}

	return image
}

func probe(app *kappv1alpha1.App, initialDelay int, failureThreshold int) *corev1.Probe {
	port := intstr.FromInt(int(*app.Spec.Port))
	if app.Spec.HealthCheckType == "tcp" {
		return &corev1.Probe{
			FailureThreshold:    int32(failureThreshold),
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			TimeoutSeconds:      1,
			InitialDelaySeconds: int32(initialDelay),
			Handler: corev1.Handler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: port,
				},
			},
		}
	} else {
		return &corev1.Probe{
			FailureThreshold:    int32(failureThreshold),
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			TimeoutSeconds:      10,
			InitialDelaySeconds: int32(initialDelay),
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Port:   port,
					Path:   app.Spec.HealthCheckEndpoint,
					Scheme: "HTTP",
				},
			},
		}
	}
}

func (r *AppReconciler) logDeploymentInequality(dep *appsv1.Deployment, propertyName string, desired, actual interface{}) {
	r.logDifference(desired, actual, propertyName, dep.Name, dep.Namespace, dep.TypeMeta)
}

func (r *AppReconciler) deploymentEquality(desired *appsv1.Deployment, actual *appsv1.Deployment) bool {

	// Validate all desired labels are present
	if !mapMatch(desired.Labels, actual.Labels) {
		r.logDeploymentInequality(desired, "labels", desired.Labels, actual.Labels)
		return false
	}

	// Validate all desired annotations are present
	if !mapMatch(desired.Annotations, actual.Annotations) {
		r.logDeploymentInequality(desired, "annotations", desired.Annotations, desired.Labels)
		return false
	}

	// Check if Replicas match
	if *actual.Spec.Replicas != *desired.Spec.Replicas {
		r.logDeploymentInequality(desired, "replicas", desired.Spec.Replicas, actual.Spec.Replicas)
		return false
	}

	// Ensure Pod Labels are set correctly
	if !mapMatch(desired.Spec.Template.Labels, actual.Spec.Template.Labels) {
		r.logDeploymentInequality(desired, "podLabels", desired.Spec.Template.Labels, actual.Spec.Template.Labels)
		return false
	}

	// Ensure Pod Annotations are set correctly
	if !mapMatch(desired.Spec.Template.Annotations, actual.Spec.Template.Annotations) {
		r.logDeploymentInequality(desired, "podAnnotations", desired.Spec.Template.Annotations, actual.Spec.Template.Annotations)
		return false
	}

	aps := actual.Spec.Template.Spec
	dps := desired.Spec.Template.Spec

	// Ensure Security Context is correct
	if *aps.SecurityContext.FSGroup != *dps.SecurityContext.FSGroup {
		r.logDeploymentInequality(desired, "securityContext", dps.SecurityContext, aps.SecurityContext)
		return false
	}

	// Ensure Service Account is correct
	if aps.ServiceAccountName != dps.ServiceAccountName {
		r.logDeploymentInequality(desired, "serviceAccountName", dps.ServiceAccountName, aps.ServiceAccountName)
		return false
	}

	// Ensure Affinity is correct
	if !reflect.DeepEqual(aps.Affinity, dps.Affinity) {
		r.logDeploymentInequality(desired, "affinity", dps.Affinity, aps.Affinity)
		return false
	}

	// Ensure Node Selector is correct
	if !reflect.DeepEqual(aps.NodeSelector, dps.NodeSelector) {
		r.logDeploymentInequality(desired, "nodeSelector", dps.NodeSelector, aps.NodeSelector)
		return false
	}

	acs := aps.Containers[0]
	dcs := dps.Containers[0]

	// Ensure Name is correct
	if acs.Name != dcs.Name {
		r.logDeploymentInequality(desired, "containerName", dcs.Name, acs.Name)
		return false
	}

	// Ensure Image is correct
	if acs.Image != dcs.Image {
		r.logDeploymentInequality(desired, "image", dcs.Image, acs.Image)
		return false
	}

	// Ensure env is correct
	for _, dv := range dcs.Env {
		found := false
		for _, av := range acs.Env {
			if av.Name == dv.Name && av.Value == dv.Value {
				found = true
				break
			}
		}
		if found == false {
			r.logDeploymentInequality(desired, "containerEnv", dcs.Env, acs.Env)
			return false
		}
	}

	// Ensure envFrom is correct
	for _, dv := range dcs.EnvFrom {
		found := false
		for _, av := range acs.EnvFrom {
			if dv.SecretRef != nil {
				if av.SecretRef != nil && av.SecretRef.Name == dv.SecretRef.Name {
					found = true
					break
				}
			}
			if dv.ConfigMapRef != nil {
				if av.ConfigMapRef != nil && av.ConfigMapRef.Name == dv.ConfigMapRef.Name {
					found = true
					break
				}
			}
		}
		if found == false {
			r.logDeploymentInequality(desired, "containerEnvFrom", dcs.EnvFrom, acs.EnvFrom)
			return false
		}
	}

	// Ensure ImagePullPolicy is correct
	if acs.ImagePullPolicy != dcs.ImagePullPolicy {
		r.logDeploymentInequality(desired, "imagePullPolicy", dcs.ImagePullPolicy, acs.ImagePullPolicy)
		return false
	}

	// Ensure ReadinessProbe is correct
	if !reflect.DeepEqual(*acs.ReadinessProbe, *dcs.ReadinessProbe) {
		r.logDeploymentInequality(desired, "readinessProbe", dcs.ReadinessProbe, acs.ReadinessProbe)
		return false
	}

	// Ensure LivenessProbe is correct
	if !reflect.DeepEqual(*acs.LivenessProbe, *dcs.LivenessProbe) {
		r.logDeploymentInequality(desired, "livenessProbe", dcs.LivenessProbe, acs.LivenessProbe)
		return false
	}

	// Ensure Ports are correct
	if !reflect.DeepEqual(acs.Ports, dcs.Ports) {
		r.logDeploymentInequality(desired, "ports", dcs.Ports, acs.Ports)
		return false
	}

	// Ensure Resource Quotas are correct
	if !reflect.DeepEqual(acs.Resources, dcs.Resources) {
		r.logDeploymentInequality(desired, "resources", dcs.Resources, acs.Resources)
		return false
	}

	// Ensure Container Security Context is correct
	if !reflect.DeepEqual(acs.SecurityContext, dcs.SecurityContext) {
		r.logDeploymentInequality(desired, "containerSecurityContext", dcs.SecurityContext, acs.SecurityContext)
		return false
	}

	return true
}
