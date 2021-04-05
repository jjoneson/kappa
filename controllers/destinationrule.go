package controllers

import (
	"context"
	"fmt"
	kappv1alpha1 "github.com/jjoneson/kappa/api/v1alpha1"
	"istio.io/api/networking/v1alpha3"
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *AppReconciler) reconcileDestinationRule(ctx context.Context, req ctrl.Request, app *kappv1alpha1.App) (ctrl.Result, error) {
	found := &istio.DestinationRule{}
	desired := r.destinationRule(app)
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
			r.Log.Info("Created new DestinationRule", "Name", app.Name, "Namespace", app.Namespace)
			return ctrl.Result{Requeue: true}, nil
		}
	} else {
		return ctrl.Result{}, err
	}

	if !reflect.DeepEqual(desired.Spec, found.Spec) {
		desired.DeepCopyInto(found)
		r.Log.Info("Updating DestinationRule", "Name", app.Name, "Namespace", app.Namespace)
		if err := r.Update(ctx, found); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *AppReconciler) destinationRule(app *kappv1alpha1.App) *istio.DestinationRule {
	tlsMode := v1alpha3.ClientTLSSettings_ISTIO_MUTUAL
	if app.Spec.DisableMtls != nil && *app.Spec.DisableMtls == true {
		tlsMode = v1alpha3.ClientTLSSettings_DISABLE
	}

	return &istio.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    app.Labels,
		},
		Spec: v1alpha3.DestinationRule{
			Host: fmt.Sprintf("%s.%s.svc.cluster.local", app.Name, app.Namespace),
			TrafficPolicy: &v1alpha3.TrafficPolicy{
				Tls: &v1alpha3.ClientTLSSettings{
					Mode: tlsMode,
				},
				LoadBalancer: &v1alpha3.LoadBalancerSettings{
					LbPolicy: &v1alpha3.LoadBalancerSettings_Simple{
						Simple: v1alpha3.LoadBalancerSettings_ROUND_ROBIN,
					},
				},
			},
		},
	}
}
