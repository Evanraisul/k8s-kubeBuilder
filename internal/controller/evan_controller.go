/*
Copyright 2024.

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

package controller

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	webappv1 "my.domain.com/k8s-kubeBuilder/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
)

// EvanReconciler reconciles a Evan object
type EvanReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=webapp.my.domain.com,resources=evans,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=webapp.my.domain.com,resources=evans/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=webapp.my.domain.com,resources=evans/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Evan object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile

func generateDeploymentName(evanName string, evanDeploymentName string, resourceCreationTimestamp int64) string {
	deploymentName := fmt.Sprintf("%s-%s-%s", evanName, evanDeploymentName, strconv.FormatInt(resourceCreationTimestamp, 10))
	if evanDeploymentName == "" {
		deploymentName = fmt.Sprintf("%s-%s", evanName, strconv.FormatInt(resourceCreationTimestamp, 10))
	}
	return deploymentName
}
func generateServiceName(evanName string, evanServiceName string, resourceCreationTimestamp int64) string {
	serviceName := fmt.Sprintf("%s-%s-%s", evanName, evanServiceName, strconv.FormatInt(resourceCreationTimestamp, 10))
	if evanServiceName == "" {
		serviceName = fmt.Sprintf("%s-%s", evanName, strconv.FormatInt(resourceCreationTimestamp, 10))
	}
	return serviceName
}

func isDeploymentConfigChanged(evanReplicas int32, evanDeploymentName string, evanDeploymentImage string, deletionPolicy webappv1.DeletionPolicy, currentDeployment *appsv1.Deployment) bool {
	if evanReplicas != 0 && evanReplicas != *currentDeployment.Spec.Replicas {
		*currentDeployment.Spec.Replicas = evanReplicas
		//fmt.Println("1")
		return true
	}
	if evanDeploymentName != "" && evanDeploymentName != currentDeployment.Name {
		currentDeployment.Name = evanDeploymentName
		//fmt.Println("2")
		return true
	}
	if evanDeploymentImage != "" && evanDeploymentImage != currentDeployment.Spec.Template.Spec.Containers[0].Image {
		currentDeployment.Spec.Template.Spec.Containers[0].Image = evanDeploymentImage
		//fmt.Println("3")
		return true
	}
	return false
}

func isServiceConfigChanged(evanServiceName string, evanServicePort int32, deletionPolicy webappv1.DeletionPolicy, currentService *corev1.Service) bool {
	if evanServiceName != "" && evanServiceName != currentService.Name {
		currentService.Name = evanServiceName
		return true
	}
	if evanServicePort != 0 && evanServicePort != currentService.Spec.Ports[0].Port {
		currentService.Spec.Ports[0].Port = evanServicePort
		return true
	}
	return false
}
func (r *EvanReconciler) deleteExternalResources(ctx context.Context, evan webappv1.Evan, deployment appsv1.Deployment, service corev1.Service) error {
	err := r.Client.Delete(ctx, &deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Println("Deployment not found")
			return nil
		}
		//return err
	}
	fmt.Println("Deployment deleted")
	//--------------------------------------------------
	err = r.Client.Delete(ctx, &service)
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Println("Service not found")
			return nil
		}
		//return err
	}
	fmt.Println("Service deleted")
	return nil
}

func (r *EvanReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log.FromContext(ctx).WithValues("ReqName", req.Name, "ReqNamespace", req.Namespace)

	// TODO(user): your logic here

	// # Load The Evan by Name
	// We'll fetch the Evan using our client.  All client methods take a
	// context (to allow for cancellation) as their first argument, and the object
	// in question as their last.  Get is a bit special, in that it takes a
	// [`NamespacedName`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client?tab=doc#ObjectKey)
	// as the middle argument (most don't have a middle argument, as we'll see
	// below).
	var evan webappv1.Evan
	var deploymentInstance appsv1.Deployment
	var serviceInstance corev1.Service

	if err := r.Get(ctx, req.NamespacedName, &evan); err != nil {
		log.Log.Info("Unable to get Evan")
		// we'll ignore not-found errors, since they can't be fixed by an immediate requeue
		// (Need to wait for a new notification), and we can get them on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	fmt.Println("1")

	// name of our custom finalizer
	myFinalizer := "Evan"
	// examine DeletionTimestamp to determine if object is under deletion
	if evan.ObjectMeta.DeletionTimestamp.IsZero() {
		fmt.Println("2")
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// to registering our finalizer.
		if !controllerutil.ContainsFinalizer(&evan, myFinalizer) {
			fmt.Println("3")
			controllerutil.AddFinalizer(&evan, myFinalizer)
			fmt.Println("4")
			if err := r.Client.Update(ctx, &evan); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	} else {
		// The object is being deleted
		fmt.Println("2")
		if controllerutil.ContainsFinalizer(&evan, myFinalizer) {
			fmt.Println("3")
			// our finalizer is present, so lets handle any external dependency
			if evan.Spec.DeletionPolicy == "WipeOut" {
				if err := r.deleteExternalResources(ctx, evan, deploymentInstance, serviceInstance); err != nil {
					fmt.Println("4")
					// if fail to delete the external dependency here, return with error
					// so that it can be retried.
					return ctrl.Result{}, err
				}
				fmt.Println("5")
				// remove our finalizer from the list and update it.
				controllerutil.RemoveFinalizer(&evan, myFinalizer)
				fmt.Println("6")
				fmt.Println("8")
				return ctrl.Result{}, nil
			}
			if err := r.Client.Update(ctx, &evan); err != nil {
				fmt.Println("7")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		fmt.Println("9")
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// Get Resource CreationTimestamp
	resourceCreationTimestamp := evan.CreationTimestamp.Unix()
	// Deployment Name
	deploymentName := generateDeploymentName(evan.Name, evan.Spec.DeploymentConfig.Name, resourceCreationTimestamp)
	// Get the deployment with the name specified in Evan.spec

	//fmt.Println("Evan1")
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: evan.Namespace, Name: deploymentName}, &deploymentInstance); err != nil {
		if errors.IsNotFound(err) {
			log.Log.Info("Could not find existing deployment")
			deploymentInstance = *newDeployment(&evan, deploymentName)
			if err := r.Client.Create(ctx, &deploymentInstance); err != nil {
				log.Log.Info("Error while creating deployment")
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
			log.Log.Info("Deployment Created")
		}
		//log.Log.Error(err, "Error fetching deployment")
		//return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	//fmt.Println("Evan2")

	// If deployment config changed, then update the Deployment.
	deletionPolicy := evan.Spec.DeletionPolicy
	replicas := evan.Spec.DeploymentConfig.Replicas
	image := evan.Spec.DeploymentConfig.Image

	if isDeploymentConfigChanged(*replicas, deploymentName, image, deletionPolicy, &deploymentInstance) {
		log.Log.Info("Deployment Config changed... Updating")
		if err := r.Client.Update(ctx, &deploymentInstance); err != nil {
			log.Log.Info("Error Updating Deployment")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		log.Log.Info("Deployment Updated.")
	}

	// Service Name
	serviceName := generateServiceName(evan.Name, evan.Spec.ServiceConfig.Name, resourceCreationTimestamp)
	// Get the service port
	servicePort := evan.Spec.ServiceConfig.Port
	if servicePort == 0 {
		log.Log.Info("Service Port is not provided by user")
		err := fmt.Errorf("Service Port is not provided by user")
		return ctrl.Result{}, err
	}
	//fmt.Println("Evan3")
	// If TargetPort is not defined by User, set the TargetPort as same as Port
	//if evan.Spec.ServiceConfig.TargetPort == 0 {
	//	serviceInstance.Spec.Ports[0].TargetPort = intstr.FromInt32(servicePort)
	//}

	//fmt.Println("Evan4")
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: evan.Namespace, Name: serviceName}, &serviceInstance); err != nil {
		if errors.IsNotFound(err) {
			log.Log.Info("Could not find existing Service")
			serviceInstance = *newService(&evan, serviceName, servicePort)

			if err := r.Client.Create(ctx, &serviceInstance); err != nil {
				log.Log.Error(err, "Error while creating Service")
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
			log.Log.Info("Service Created")
		}
		//		log.Log.Error(err, "Error fetching Service")
		//return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	//fmt.Println("Evan5..")

	// If ServiceConfig changes, update the service
	if isServiceConfigChanged(serviceName, servicePort, deletionPolicy, &serviceInstance) {
		log.Log.Info("Service Config changed... Updating")
		if err := r.Client.Update(ctx, &serviceInstance); err != nil {
			log.Log.Error(err, "Error Updating Service")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		log.Log.Info("Service Updated Successfully.")
	}

	// Update the Status of the Evan
	err := r.updateevan(ctx, &evan, &deploymentInstance)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	return ctrl.Result{}, nil
}

func (r *EvanReconciler) updateevan(ctx context.Context, Evan *webappv1.Evan, deployment *appsv1.Deployment) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	EvanCopy := Evan.DeepCopy()
	EvanCopy.Status.AvailableReplicas = deployment.Status.AvailableReplicas
	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Evan resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	err := r.Client.Status().Update(ctx, EvanCopy)
	return err
}

var (
	deployOwnerKey = ".metadata.controller"
	svcOwnerKey    = ".metadata.controller"
	apiGVStr       = webappv1.GroupVersion.String()
	ourKind        = "Evan"
)

// SetupWithManager sets up the controller with the Manager.
func (r *EvanReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// For Deployment
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &appsv1.Deployment{}, deployOwnerKey, func(object client.Object) []string {
		// Grab the Deployment object
		deployment := object.(*appsv1.Deployment)
		// Extract the owner
		owner := metav1.GetControllerOf(deployment)
		if owner == nil {
			return nil
		}
		// Make sure it is a Custom Resource
		if owner.APIVersion != apiGVStr || owner.Kind != ourKind {
			return nil
		}
		// ...and if so, return it.
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	// For Service
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Service{}, svcOwnerKey, func(object client.Object) []string {
		// Grab the Service
		service := object.(*corev1.Service)
		// Extract the owner
		owner := metav1.GetControllerOf(service)
		if owner == nil {
			return nil
		}
		// Make sure it is a Custom Resource
		if owner.APIVersion != apiGVStr || owner.Kind != ourKind {
			return nil
		}
		// ...and if so, return it.
		return []string{}
	}); err != nil {
		return err
	}

	// Watches and Custom EventHandler
	/*handlerForDeployment := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []reconcile.Request {
	// List all Custom Resource
	customResources := &webappv1.EvanList{}
	if err := r.List(context.Background(), customResources); err != nil {
		return nil
	}
	// This func return a Reconcile Request Array
	var req []reconcile.Request
	for _, c := range customResources.Items {
		deploymentName := func() string {
			return generateDeploymentName(c.Name, c.Spec.DeploymentConfig.Name, c.CreationTimestamp.Unix())
		}()
		// Find the deployment owned by the CR
		if deploymentName == object.GetName() && c.Namespace == object.GetNamespace() {
			deploy := &appsv1.Deployment{}
			if err := r.Get(context.Background(), types.NamespacedName{
				Namespace: object.GetNamespace(),
				Name:      object.GetName(),
			}, deploy); err != nil {
				// This case can happen if somehow deployment gets deleted by
				// Kubectl command. We need to append new reconcile request to array
				// to create desired number of deployment again.
				if errors.IsNotFound(err) {
					req = append(req, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: c.Namespace,
							Name:      c.Name,
						},
					})
					continue
				} else {
					return nil
				}
			}
			// Only append to the reconcile request array if replica count miss match.
			if deploy.Spec.Replicas != c.Spec.DeploymentConfig.Replicas {
				req = append(req, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: c.Namespace,
						Name:      c.Name,
					},
				})
			}
		}
	}
	*/
	//return req

	fmt.Println("SetupWithManager Successful.")

	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.Evan{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

// newDeployment creates a new Deployment for an Evan resource. It also sets
// the appropriate OwnerReferences on the resource so SetupWithManager can discover
// the Evan resource that 'owns' it.
func newDeployment(Evan *webappv1.Evan, deploymentName string) *appsv1.Deployment {

	deployment := &appsv1.Deployment{}
	labels := map[string]string{
		"app": "my-book",
	}
	deployment.Labels = labels
	deployment.TypeMeta.Kind = "Deployment"

	deployment.ObjectMeta.Name = deploymentName
	deployment.ObjectMeta.Namespace = Evan.ObjectMeta.Namespace
	deployment.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(Evan, webappv1.GroupVersion.WithKind("Evan")),
	}

	deployment.Spec.Replicas = Evan.Spec.DeploymentConfig.Replicas
	deployment.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: labels,
	}
	deployment.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Labels: labels,
	}
	deployment.Spec.Template.Spec = corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "my-book",
				Image: Evan.Spec.DeploymentConfig.Image,
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: Evan.Spec.ServiceConfig.Port,
					},
				},
			},
		},
	}
	return deployment
}

// newService creates a new Service for an Evan resource. It also sets
// the appropriate OwnerReferences on the resource so SetupWithManager can discover
// the Evan resource that 'owns' it.
func newService(Evan *webappv1.Evan, serviceName string, serviceTargetPort int32) *corev1.Service {

	labels := map[string]string{
		"app": "my-book",
	}
	service := &corev1.Service{}
	service.Labels = labels

	service.TypeMeta = metav1.TypeMeta{
		Kind: "Service",
	}

	service.ObjectMeta = metav1.ObjectMeta{
		Name:      serviceName,
		Namespace: Evan.ObjectMeta.Namespace,
	}
	service.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(Evan, webappv1.GroupVersion.WithKind("Evan")),
	}

	service.Spec = corev1.ServiceSpec{
		Type:     Evan.Spec.ServiceConfig.Type,
		Selector: labels,
	}
	service.Spec.Ports = []corev1.ServicePort{
		corev1.ServicePort{
			Port:       Evan.Spec.ServiceConfig.Port,
			TargetPort: intstr.FromInt32(serviceTargetPort),
			NodePort:   Evan.Spec.ServiceConfig.NodePort,
		},
	}
	return service
}
