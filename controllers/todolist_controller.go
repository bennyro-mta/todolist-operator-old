package controllers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	todolistv1 "todolist-operator/api/v1"
)

// TodoListReconciler reconciles a TodoList object
type TodoListReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=todolist.example.com,resources=todolists,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=todolist.example.com,resources=todolists/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=todolist.example.com,resources=todolists/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *TodoListReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	todoList := &todolistv1.TodoList{}
	if err := r.Get(ctx, req.NamespacedName, todoList); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	owner := todoList.Spec.Owner
	if owner == "" {
		logger.Info("TodoList spec.owner is empty, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// Always reconcile resources
	if err := r.reconcileSecret(ctx, todoList, owner); err != nil {
		logger.Error(err, "failed to reconcile secret")
		todoList.Status.Phase = "Failed"
		_ = r.Status().Update(ctx, todoList)
		return ctrl.Result{}, err
	}
	if err := r.reconcileConfigMaps(ctx, todoList, owner); err != nil {
		logger.Error(err, "failed to reconcile configmaps")
		todoList.Status.Phase = "Failed"
		_ = r.Status().Update(ctx, todoList)
		return ctrl.Result{}, err
	}
	if err := r.reconcileMariaDB(ctx, todoList, owner); err != nil {
		logger.Error(err, "failed to reconcile mariadb")
		todoList.Status.Phase = "Failed"
		_ = r.Status().Update(ctx, todoList)
		return ctrl.Result{}, err
	}
	if err := r.reconcileTodoAPI(ctx, todoList, owner); err != nil {
		logger.Error(err, "failed to reconcile todo-api")
		todoList.Status.Phase = "Failed"
		_ = r.Status().Update(ctx, todoList)
		return ctrl.Result{}, err
	}
	if err := r.reconcileFrontend(ctx, todoList, owner); err != nil {
		logger.Error(err, "failed to reconcile frontend")
		todoList.Status.Phase = "Failed"
		_ = r.Status().Update(ctx, todoList)
		return ctrl.Result{}, err
	}

	// Update status to Running after all reconciliations succeed
	todoList.Status.Phase = "Running"
	todoList.Status.MariaDBReady = true
	todoList.Status.TodoAPIReady = true
	todoList.Status.FrontendReady = true
	if err := r.Status().Update(ctx, todoList); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	logger.Info("TodoList reconciled successfully", "owner", owner, "namespace", todoList.Namespace)
	return ctrl.Result{}, nil
}

func (r *TodoListReconciler) reconcileSecret(ctx context.Context, todoList *todolistv1.TodoList, owner string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-todolist-secret", owner),
			Namespace: todoList.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"MYSQL_ROOT_PASSWORD": []byte("todolist123"),
		},
	}
	if err := controllerutil.SetControllerReference(todoList, secret, r.Scheme); err != nil {
		return err
	}

	found := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Name: secret.Name, Namespace: secret.Namespace}, found); err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, secret)
		}
		return err
	}
	return nil
}

func (r *TodoListReconciler) reconcileConfigMaps(ctx context.Context, todoList *todolistv1.TodoList, owner string) error {
	apiCfg := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-todolist-config", owner),
			Namespace: todoList.Namespace,
		},
		Data: map[string]string{
			"MYSQL_DB":    "todolist",
			"MYSQL_HOST":  fmt.Sprintf("%s-mariadb", owner),
			"MYSQL_USER":  "root",
			"MYSQL_TABLE": "todos",
		},
	}
	if err := controllerutil.SetControllerReference(todoList, apiCfg, r.Scheme); err != nil {
		return err
	}
	found := &corev1.ConfigMap{}
	if err := r.Get(ctx, client.ObjectKey{Name: apiCfg.Name, Namespace: apiCfg.Namespace}, found); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, apiCfg); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	frontCfg := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-todolist-vue-config", owner),
			Namespace: todoList.Namespace,
		},
		Data: map[string]string{
			"API_BASE_URL": "/todos",
			"USER":         owner,
		},
	}
	if err := controllerutil.SetControllerReference(todoList, frontCfg, r.Scheme); err != nil {
		return err
	}
	if err := r.Get(ctx, client.ObjectKey{Name: frontCfg.Name, Namespace: frontCfg.Namespace}, found); err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, frontCfg)
		}
		return err
	}
	return nil
}

func (r *TodoListReconciler) reconcileMariaDB(ctx context.Context, todoList *todolistv1.TodoList, owner string) error {
	replicas := int32(1)
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-mariadb", owner),
			Namespace: todoList.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "mariadb", "owner": owner},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "mariadb", "owner": owner},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "mariadb",
							Image: "mariadb:latest",
							Ports: []corev1.ContainerPort{{ContainerPort: 3306}},
							Env: []corev1.EnvVar{
								{
									Name: "MYSQL_ROOT_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: fmt.Sprintf("%s-todolist-secret", owner)},
											Key:                  "MYSQL_ROOT_PASSWORD",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "mariadb-data", MountPath: "/var/lib/mysql"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{Name: "mariadb-data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
					},
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(todoList, deploy, r.Scheme); err != nil {
		return err
	}
	found := &appsv1.Deployment{}
	if err := r.Get(ctx, client.ObjectKey{Name: deploy.Name, Namespace: deploy.Namespace}, found); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, deploy); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-mariadb", owner),
			Namespace: todoList.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "mariadb", "owner": owner},
			Ports:    []corev1.ServicePort{{Port: 3306, TargetPort: intstr.FromInt(3306)}},
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
	if err := controllerutil.SetControllerReference(todoList, svc, r.Scheme); err != nil {
		return err
	}
	foundSvc := &corev1.Service{}
	if err := r.Get(ctx, client.ObjectKey{Name: svc.Name, Namespace: svc.Namespace}, foundSvc); err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, svc)
		}
		return err
	}
	return nil
}

func (r *TodoListReconciler) reconcileTodoAPI(ctx context.Context, todoList *todolistv1.TodoList, owner string) error {
	// Get replicas from spec, default to 1
	replicas := int32(1)
	if todoList.Spec.APIReplicas != nil {
		replicas = *todoList.Spec.APIReplicas
	}

	// Get service type from spec, default to ClusterIP
	serviceType := corev1.ServiceTypeClusterIP
	if todoList.Spec.ServiceType != nil {
		serviceType = *todoList.Spec.ServiceType
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-todo-api", owner),
			Namespace: todoList.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "todo-api", "owner": owner},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "todo-api", "owner": owner},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "todo-api",
							Image: "ghcr.io/bennyro-mta/todos-api:1.2",
							Ports: []corev1.ContainerPort{{ContainerPort: 8080, Name: "http"}},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{Name: fmt.Sprintf("%s-todolist-config", owner)},
									},
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "MYSQL_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: fmt.Sprintf("%s-todolist-secret", owner)},
											Key:                  "MYSQL_ROOT_PASSWORD",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(todoList, deploy, r.Scheme); err != nil {
		return err
	}
	found := &appsv1.Deployment{}
	if err := r.Get(ctx, client.ObjectKey{Name: deploy.Name, Namespace: deploy.Namespace}, found); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, deploy); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		// Update existing deployment if replicas changed
		if found.Spec.Replicas == nil || *found.Spec.Replicas != replicas {
			found.Spec.Replicas = &replicas
			if err := r.Update(ctx, found); err != nil {
				return err
			}
		}
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-todo-api", owner),
			Namespace: todoList.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "todo-api", "owner": owner},
			Ports:    []corev1.ServicePort{{Name: "http", Port: 8080, TargetPort: intstr.FromInt(8080)}},
			Type:     serviceType,
		},
	}
	if err := controllerutil.SetControllerReference(todoList, svc, r.Scheme); err != nil {
		return err
	}
	foundSvc := &corev1.Service{}
	if err := r.Get(ctx, client.ObjectKey{Name: svc.Name, Namespace: svc.Namespace}, foundSvc); err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, svc)
		}
		return err
	} else {
		// Update existing service if type changed
		if foundSvc.Spec.Type != serviceType {
			foundSvc.Spec.Type = serviceType
			if err := r.Update(ctx, foundSvc); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *TodoListReconciler) reconcileFrontend(ctx context.Context, todoList *todolistv1.TodoList, owner string) error {
	// Get replicas from spec, default to 1
	replicas := int32(1)
	if todoList.Spec.FrontendReplicas != nil {
		replicas = *todoList.Spec.FrontendReplicas
	}

	// Get service type from spec, default to ClusterIP
	serviceType := corev1.ServiceTypeClusterIP
	if todoList.Spec.ServiceType != nil {
		serviceType = *todoList.Spec.ServiceType
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-todolist-vue", owner),
			Namespace: todoList.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "todolist-vue", "owner": owner},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "todolist-vue", "owner": owner},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "todolist-vue",
							Image: "ghcr.io/bennyro-mta/todolist-vue:1.2",
							Ports: []corev1.ContainerPort{{ContainerPort: 8080}},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{Name: fmt.Sprintf("%s-todolist-vue-config", owner)},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(todoList, deploy, r.Scheme); err != nil {
		return err
	}
	found := &appsv1.Deployment{}
	if err := r.Get(ctx, client.ObjectKey{Name: deploy.Name, Namespace: deploy.Namespace}, found); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, deploy); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		// Update existing deployment if replicas changed
		if found.Spec.Replicas == nil || *found.Spec.Replicas != replicas {
			found.Spec.Replicas = &replicas
			if err := r.Update(ctx, found); err != nil {
				return err
			}
		}
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-todolist-vue", owner),
			Namespace: todoList.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "todolist-vue", "owner": owner},
			Ports:    []corev1.ServicePort{{Name: "http", Port: 8080, TargetPort: intstr.FromInt(8080)}},
			Type:     serviceType,
		},
	}
	if err := controllerutil.SetControllerReference(todoList, svc, r.Scheme); err != nil {
		return err
	}
	foundSvc := &corev1.Service{}
	if err := r.Get(ctx, client.ObjectKey{Name: svc.Name, Namespace: svc.Namespace}, foundSvc); err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, svc)
		}
		return err
	} else {
		// Update existing service if type changed
		if foundSvc.Spec.Type != serviceType {
			foundSvc.Spec.Type = serviceType
			if err := r.Update(ctx, foundSvc); err != nil {
				return err
			}
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TodoListReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&todolistv1.TodoList{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
