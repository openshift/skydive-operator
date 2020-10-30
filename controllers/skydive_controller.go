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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/openshift/skydive-operator/api/v1alpha1"
)

// SkydiveReconciler reconciles a Skydive object
type SkydiveReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=config.skydive.network,resources=skydives,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=config.skydive.network,resources=skydives/status,verbs=get;update;patch

func (r *SkydiveReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("skydive", req.NamespacedName)

	skydive := &configv1alpha1.Skydive{}
	err := r.Get(ctx, req.NamespacedName, skydive)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			log.Info("Skydive resource not found.")

			// TODO create finalizer that cleans up
			// TODO even if finalizer is supposed to clean up, check here too

			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Skydive resource")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *SkydiveReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&configv1alpha1.Skydive{}).
		Complete(r)
}
