package controllers

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func mapMatch(desired map[string]string, actual map[string]string) bool {
	for k, v := range desired {
		if _, ok := actual[k]; !ok {
			return false
		} else if actual[k] != v {
			return false
		}
	}
	return true
}

func (r *AppReconciler) logDifference(desired, actual interface{}, propertyName, name, namespace string, meta metav1.TypeMeta) {
	r.Log.Info("Updating mismatched values", "type", meta, "name", name, "namespace", namespace, "propertyName", propertyName, "desired", desired, "actual", actual)
}
