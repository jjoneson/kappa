package controllers

import (
	"context"
	kappv1alpha1 "github.com/jjoneson/kappa/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *AppReconciler) reconcileServiceAccount(ctx context.Context, req ctrl.Request, app *kappv1alpha1.App) (ctrl.Result, error) {
	found := &corev1.ServiceAccount{}
	desired := r.serviceAccount(app)

	err := r.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = r.Create(ctx, desired); err != nil {
				return ctrl.Result{}, err
			}
			r.Log.Info("Created new ServiceAccount", "Name", app.Name, "Namespace", app.Namespace)
			return ctrl.Result{Requeue: true}, nil
		} else {
			return ctrl.Result{}, err
		}
	}

	desired.DeepCopyInto(found)
	r.Log.Info("Updating ServiceAccount", "Name", app.Name, "Namespace", app.Namespace)
	if err := r.Update(ctx, found); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *AppReconciler) serviceAccount(app *kappv1alpha1.App) *corev1.ServiceAccount {
	labels := make(map[string]string)
	annotations := make(map[string]string)
	if app.Spec.Labels != nil {
		for k, v := range app.Spec.Labels {
			labels[k] = v
		}
	}

	if app.Spec.Annotations != nil {
		for k, v := range app.Spec.Annotations {
			annotations[k] = v
		}
	}

	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        app.Name,
			Namespace:   app.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
	}
}
