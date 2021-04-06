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

package controllers

import (
	//	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	//	"reflect"

	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cachev1alpha1 "github.com/soudrisaivishnu/kubernetes-operators/api/v1alpha1"
)

// WxccOperatorReconciler reconciles a WxccOperator object
type WxccOperatorReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cache.example.com,resources=wxccoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.example.com,resources=wxccoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cache.example.com,resources=wxccoperators/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WxccOperator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile

func (r *WxccOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("wxccoperator", req.NamespacedName)

	// Calling the wxcc operator instance
	wxcc := &cachev1alpha1.WxccOperator{}
	err := r.Get(ctx, req.NamespacedName, wxcc)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("wxcc resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get wxcc")
		return ctrl.Result{}, err
	}

	// config map code
	configMapChanged, err := r.ensureLatestConfigMap(wxcc)
	if err != nil {
		return ctrl.Result{}, err
	}

	if configMapChanged {
		log.Info("Config map already exists")
	}
	return ctrl.Result{}, nil

}

func (r *WxccOperatorReconciler) ensureLatestConfigMap(wxcc *cachev1alpha1.WxccOperator) (bool, error) {

	configMap := newConfigMap(wxcc)

	// Set wxcc instance as the owner and controller
	if err := controllerutil.SetControllerReference(wxcc, configMap, r.Scheme); err != nil {
		return false, err
	}

	// Check if this ConfigMap already exists
	foundMap := &corev1.ConfigMap{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, foundMap)
	if err != nil && errors.IsNotFound(err) {
		err = r.Create(context.TODO(), configMap)
		if err != nil {
			return false, err
		}
	} else if err != nil {
		return false, err
	}

	if foundMap.Data["config.md"] != configMap.Data["config.md"] {
		err = r.Update(context.TODO(), configMap)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func newConfigMap(cr *cachev1alpha1.WxccOperator) *corev1.ConfigMap {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-config",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"config.md": cr.Spec.Markdown,
		},
	}
}

// labelsForWxcc returns the labels for selecting the resources
// belonging to the given wxcc CR name.
func labelsForWxcc(name string) map[string]string {
	return map[string]string{"app": "wxcc", "wxcc_cr": name}
}

// SetupWithManager sets up the controller with the Manager.
func (r *WxccOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.WxccOperator{}).
		Complete(r)
}
