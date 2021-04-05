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

func (r *AppReconciler) reconcileVirtualService(ctx context.Context, req ctrl.Request, app *kappv1alpha1.App) (ctrl.Result, error) {
	found := &istio.VirtualService{}
	desired := r.virtualservice(app)
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
			r.Log.Info("Created new VirtualService", "Name", app.Name, "Namespace", app.Namespace)
			return ctrl.Result{Requeue: true}, nil
		}
	} else {
		return ctrl.Result{}, err
	}

	if !reflect.DeepEqual(desired.Spec, found.Spec) {
		desired.DeepCopyInto(found)
		r.Log.Info("Updating VirtualService", "Name", app.Name, "Namespace", app.Namespace)
		if err := r.Update(ctx, found); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *AppReconciler) virtualservice(app *kappv1alpha1.App) *istio.VirtualService {
	return &istio.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    app.Labels,
		},
		Spec: v1alpha3.VirtualService{
			Http: []*v1alpha3.HTTPRoute{
				{
					Match: []*v1alpha3.HTTPMatchRequest{
						{
							Uri: &v1alpha3.StringMatch{
								MatchType: &v1alpha3.StringMatch_Prefix{
									Prefix: "/",
								},
							},
						},
					},
					Route: []*v1alpha3.HTTPRouteDestination{
						{
							Destination: &v1alpha3.Destination{
								Host: fmt.Sprintf("%s.%s.svc.cluster.local", app.Name, app.Namespace),
								Port: &v1alpha3.PortSelector{
									Number: 80,
								},
							},
						},
					},
					Headers: &v1alpha3.Headers{
						Response: &v1alpha3.Headers_HeaderOperations{
							Set: map[string]string{"Strict-Transport-Security": "max-age=31536000"},
							Remove: []string{
								"Server",
								"server",
							},
						},
					},
					CorsPolicy: &v1alpha3.CorsPolicy{
						AllowOrigins: []*v1alpha3.StringMatch{
							{
								MatchType: &v1alpha3.StringMatch_Regex{
									Regex: ".*",
								},
							},
						},
						AllowMethods: []string{
							"GET",
							"POST",
							"PUT",
							"DELETE",
							"PATCH",
							"HEAD",
							"OPTIONS",
						},
						AllowHeaders: []string{
							"*",
						},
					},
				},
			},
		},
	}
}
