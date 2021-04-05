package controllers

import (
	"context"
	kappv1alpha1 "github.com/jjoneson/kappa/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *AppReconciler) reconcileService(ctx context.Context, req ctrl.Request, app *kappv1alpha1.App) (ctrl.Result, error) {
	found := &corev1.Service{}
	desired := r.service(app)
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
			r.Log.Info("Created new Service", "Name", app.Name, "Namespace", app.Namespace)
			return ctrl.Result{Requeue: true}, nil
		}
	} else {
		return ctrl.Result{}, err
	}

	if !r.serviceEquality(desired, found) {
		desired.DeepCopyInto(found)
		r.Log.Info("Updating deployment", "Name", app.Name, "Namespace", app.Namespace)
		if err := r.Update(ctx, found); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *AppReconciler) service(app *kappv1alpha1.App) *corev1.Service {
	labels := make(map[string]string)
	for k, v := range app.Labels {
		labels[k] = v
	}
	labels["app"] = app.Name

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        app.Name,
			Namespace:   app.Namespace,
			Labels:      labels,
			Annotations: app.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       *app.Spec.Port,
					Name:       "http",
					Protocol:   "TCP",
					TargetPort: intstr.FromInt(int(*app.Spec.Port)),
				},
				{
					Port:       80,
					Name:       "http-80",
					Protocol:   "TCP",
					TargetPort: intstr.FromInt(int(*app.Spec.Port)),
				},
			},
			Selector: map[string]string{
				"app": app.Name,
			},
			SessionAffinity: corev1.ServiceAffinityNone,
			Type:            corev1.ServiceTypeClusterIP,
		},
	}
}

func (r *AppReconciler) logServiceEquality(svc *corev1.Service, propertyName string, desired, actual interface{}) {
	r.logDifference(desired, actual, propertyName, svc.Name, svc.Namespace, svc.TypeMeta)
}

func (r *AppReconciler) serviceEquality(desired *corev1.Service, actual *corev1.Service) bool {
	// Validate all desired labels are present
	if !mapMatch(desired.Labels, actual.Labels) {
		r.logServiceEquality(desired, "labels", desired.Labels, actual.Labels)
		return false
	}

	// Validate all desired annotations are present
	if !mapMatch(desired.Annotations, actual.Annotations) {
		r.logServiceEquality(desired, "annotations", desired.Annotations, desired.Labels)
		return false
	}

	// Validate ports are correct
	for _, dport := range desired.Spec.Ports {
		found := false
		for _, aport := range actual.Spec.Ports {
			if reflect.DeepEqual(aport, dport) {
				found = true
				break
			}
		}
		if found == false {
			r.logServiceEquality(desired, "ports", desired.Annotations, desired.Labels)
			return false
		}
	}

	return true
}
