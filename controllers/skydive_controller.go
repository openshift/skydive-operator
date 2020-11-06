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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

	// TODO There should only be one of these in a cluster
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

	// Check for the Skydive analyzer ConfigMap
	// TODO If the ConfigMap exists, make sure it's current
	curAnalyzerCM := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: "skydive-analyzer-config", Namespace: skydive.Namespace}, curAnalyzerCM)
	if err != nil && errors.IsNotFound(err) {
		cm, err := r.analyzerConfig(skydive)
		if err != nil {
			log.Error(err, "Failed to create new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
			return ctrl.Result{}, err
		}
		log.Info("Creating a new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
		err = r.Create(ctx, cm)
		if err != nil {
			log.Error(err, "Failed to create new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
			return ctrl.Result{}, err
		}
		// ConfigMap created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil

	} else if err != nil {
		log.Error(err, "Failed to get Analyzer ConfigMap")
		return ctrl.Result{}, err
	}

	// Check for the skydive analyzer deployment
	// TODO If the deployment exists, make sure it's current
	curDep := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: "skydive-analyzer", Namespace: skydive.Namespace}, curDep)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep, err := r.analyzerDeployment(skydive)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// Check for the Skydive agent ConfigMap
	// TODO If the ConfigMap exists, make sure it's current
	curAgentCM := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: "skydive-agent-config", Namespace: skydive.Namespace}, curAgentCM)
	if err != nil && errors.IsNotFound(err) {
		cm, err := r.agentConfig(skydive)
		if err != nil {
			log.Error(err, "Failed to create new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
			return ctrl.Result{}, err
		}
		log.Info("Creating a new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
		err = r.Create(ctx, cm)
		if err != nil {
			log.Error(err, "Failed to create new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
			return ctrl.Result{}, err
		}
		// ConfigMap created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil

	} else if err != nil {
		log.Error(err, "Failed to get Agent ConfigMap")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *SkydiveReconciler) agentConfig(skydive *configv1alpha1.Skydive) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "skydive-agent-config",
			Namespace: skydive.Namespace,
			Labels:    map[string]string{"app": "skydive-agent"},
		},
		Data: map[string]string{
			"SKYDIVE_AGENT_TOPOLOGY_PROBES": "netns netlink ovsdb socketinfo",
			"SKYDIVE_AGENT_LISTEN":          "127.0.0.1:8081",
		},
	}
	err := ctrl.SetControllerReference(skydive, cm, r.Scheme)
	if err != nil {
		return nil, err
	}
	return cm, nil
}

func (r *SkydiveReconciler) analyzerConfig(skydive *configv1alpha1.Skydive) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "skydive-analyzer-config",
			Namespace: skydive.Namespace,
			Labels:    map[string]string{"app": "skydive-analyzer"},
		},
		Data: map[string]string{
			"SKYDIVE_ANALYZER_FLOW_BACKEND":     "elasticsearch",
			"SKYDIVE_ANALYZER_TOPOLOGY_BACKEND": "elasticsearch",
		},
	}
	err := ctrl.SetControllerReference(skydive, cm, r.Scheme)
	if err != nil {
		return nil, err
	}
	return cm, nil
}

func (r *SkydiveReconciler) analyzerDeployment(skydive *configv1alpha1.Skydive) (*appsv1.Deployment, error) {
	replicas := int32(1)
	labels := map[string]string{"app": "skydive", "tier": "analayzer"}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "skydive-analyzer",
			Namespace: skydive.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "skydive-analyzer",
							Image: "skydive/skydive",
							Args:  []string{"analyzer"},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 8082},
								{
									ContainerPort: 8082,
									Protocol:      "UDP",
								},
								{ContainerPort: 12379},
								{ContainerPort: 12380},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "SKYDIVE_ANALYZER_LISTEN",
									Value: "0.0.0.0:8082",
								},
								{
									Name:  "SKYDIVE_ETCD_LISTEN",
									Value: "0.0.0.0:12379",
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "skydive-analyzer-config",
										},
									},
								},
							},
							// TODO Readiness probe
							// TODO Liveness probe
						},
						{
							// TODO This container needs to be dropped in favor of using
							// an existing elasticsearch that is maintained via another
							// operator.  Perhaps this can remain for dev/test usage ...
							Name:  "skydive-elasticsearch",
							Image: "elasticsearch:5",
							Args:  []string{"-E", "http.port=9200"},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 9200},
							},
						},
					},
				},
			},
		},
	}
	err := ctrl.SetControllerReference(skydive, dep, r.Scheme)
	if err != nil {
		return nil, err
	}
	return dep, nil
}

func (r *SkydiveReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&configv1alpha1.Skydive{}).
		Complete(r)
}
