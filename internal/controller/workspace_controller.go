package controller

import (
	"context"
	"fmt"

	apiv1alpha1 "github.com/surefire-ai/korus/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	tenantReadyCondition    = "Ready"
	workspaceReadyCondition = "Ready"
)

type TenantReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type WorkspaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var tenant apiv1alpha1.Tenant
	if err := r.Get(ctx, req.NamespacedName, &tenant); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	original := tenant.DeepCopy()
	previousStatus := tenant.Status.DeepCopy()

	var workspaces apiv1alpha1.WorkspaceList
	if err := r.List(ctx, &workspaces, client.InNamespace(req.Namespace)); err != nil {
		return ctrl.Result{}, err
	}

	count := int32(0)
	for _, workspace := range workspaces.Items {
		if workspace.Spec.TenantRef.Name == tenant.Name {
			count++
		}
	}

	setTenantStatus(&tenant, "Ready", count, metav1.Condition{
		Type:               tenantReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             "WorkspaceCountComputed",
		Message:            fmt.Sprintf("tenant has %d workspace(s)", count),
		ObservedGeneration: tenant.Generation,
	})

	if equality.Semantic.DeepEqual(previousStatus, &tenant.Status) {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, r.Status().Patch(ctx, &tenant, client.MergeFrom(original))
}

func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.Tenant{}).
		Complete(r)
}

func (r *WorkspaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var workspace apiv1alpha1.Workspace
	if err := r.Get(ctx, req.NamespacedName, &workspace); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	original := workspace.DeepCopy()
	previousStatus := workspace.Status.DeepCopy()

	var tenant apiv1alpha1.Tenant
	if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: workspace.Spec.TenantRef.Name}, &tenant); err != nil {
		if apierrors.IsNotFound(err) {
			setWorkspaceStatus(&workspace, "NotReady", "", resolvedWorkspaceNamespace(workspace), nil, metav1.Condition{
				Type:               workspaceReadyCondition,
				Status:             metav1.ConditionFalse,
				Reason:             "TenantReferenceFailed",
				Message:            fmt.Sprintf("referenced Tenant %q not found", workspace.Spec.TenantRef.Name),
				ObservedGeneration: workspace.Generation,
			})
			if equality.Semantic.DeepEqual(previousStatus, &workspace.Status) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, r.Status().Patch(ctx, &workspace, client.MergeFrom(original))
		}
		return ctrl.Result{}, err
	}

	endpoint := map[string]string{
		"console": fmt.Sprintf("/tenants/%s/workspaces/%s", tenant.Name, workspace.Name),
	}
	setWorkspaceStatus(&workspace, "Ready", tenant.Name, resolvedWorkspaceNamespace(workspace), endpoint, metav1.Condition{
		Type:               workspaceReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             "TenantResolved",
		Message:            "workspace resolved its tenant and console scope",
		ObservedGeneration: workspace.Generation,
	})

	if equality.Semantic.DeepEqual(previousStatus, &workspace.Status) {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, r.Status().Patch(ctx, &workspace, client.MergeFrom(original))
}

func (r *WorkspaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.Workspace{}).
		Complete(r)
}

func setTenantStatus(tenant *apiv1alpha1.Tenant, phase string, workspaceCount int32, condition metav1.Condition) {
	tenant.Status.Phase = phase
	tenant.Status.ObservedGeneration = tenant.Generation
	tenant.Status.WorkspaceCount = workspaceCount
	tenant.Status.Conditions = mergeCondition(tenant.Status.Conditions, condition)
}

func setWorkspaceStatus(workspace *apiv1alpha1.Workspace, phase string, tenantRef string, namespace string, endpoint map[string]string, condition metav1.Condition) {
	workspace.Status.Phase = phase
	workspace.Status.ObservedGeneration = workspace.Generation
	workspace.Status.TenantRef = tenantRef
	workspace.Status.Namespace = namespace
	workspace.Status.Endpoint = endpoint
	workspace.Status.Conditions = mergeCondition(workspace.Status.Conditions, condition)
}

func resolvedWorkspaceNamespace(workspace apiv1alpha1.Workspace) string {
	if workspace.Spec.Namespace != "" {
		return workspace.Spec.Namespace
	}
	return workspace.Namespace
}
