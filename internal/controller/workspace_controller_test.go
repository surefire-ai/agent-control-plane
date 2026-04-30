package controller

import (
	"context"
	"testing"

	apiv1alpha1 "github.com/surefire-ai/korus/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestWorkspaceReconcilerMarksReadyWhenTenantExists(t *testing.T) {
	scheme := testScheme(t)
	tenant := &apiv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-a", Namespace: "platform"},
	}
	workspace := &apiv1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "workspace-a",
			Namespace:  "platform",
			Generation: 2,
		},
		Spec: apiv1alpha1.WorkspaceSpec{
			TenantRef:   apiv1alpha1.LocalObjectReference{Name: "tenant-a"},
			DisplayName: "Workspace A",
			Namespace:   "team-a",
		},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.Workspace{}, &apiv1alpha1.Tenant{}).
		WithObjects(tenant, workspace).
		Build()
	reconciler := &WorkspaceReconciler{Client: kubeClient, Scheme: scheme}

	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "platform", Name: "workspace-a"}}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	var updated apiv1alpha1.Workspace
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &updated); err != nil {
		t.Fatalf("get Workspace returned error: %v", err)
	}
	if updated.Status.Phase != "Ready" {
		t.Fatalf("expected Ready phase, got %q", updated.Status.Phase)
	}
	if updated.Status.TenantRef != "tenant-a" {
		t.Fatalf("expected tenant ref, got %#v", updated.Status)
	}
	if updated.Status.Namespace != "team-a" {
		t.Fatalf("expected resolved namespace, got %#v", updated.Status)
	}
	if updated.Status.Endpoint["console"] != "/tenants/tenant-a/workspaces/workspace-a" {
		t.Fatalf("expected console endpoint, got %#v", updated.Status.Endpoint)
	}
	if len(updated.Status.Conditions) != 1 || updated.Status.Conditions[0].Reason != "TenantResolved" {
		t.Fatalf("expected TenantResolved condition, got %#v", updated.Status.Conditions)
	}
}

func TestWorkspaceReconcilerMarksNotReadyWhenTenantMissing(t *testing.T) {
	scheme := testScheme(t)
	workspace := &apiv1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "workspace-a",
			Namespace:  "platform",
			Generation: 1,
		},
		Spec: apiv1alpha1.WorkspaceSpec{
			TenantRef: apiv1alpha1.LocalObjectReference{Name: "missing-tenant"},
		},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.Workspace{}).
		WithObjects(workspace).
		Build()
	reconciler := &WorkspaceReconciler{Client: kubeClient, Scheme: scheme}

	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "platform", Name: "workspace-a"}}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	var updated apiv1alpha1.Workspace
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &updated); err != nil {
		t.Fatalf("get Workspace returned error: %v", err)
	}
	if updated.Status.Phase != "NotReady" {
		t.Fatalf("expected NotReady phase, got %q", updated.Status.Phase)
	}
	if len(updated.Status.Conditions) != 1 || updated.Status.Conditions[0].Reason != "TenantReferenceFailed" {
		t.Fatalf("expected TenantReferenceFailed condition, got %#v", updated.Status.Conditions)
	}
}

func TestTenantReconcilerCountsWorkspaces(t *testing.T) {
	scheme := testScheme(t)
	tenant := &apiv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "tenant-a",
			Namespace:  "platform",
			Generation: 3,
		},
	}
	workspaceA := &apiv1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{Name: "workspace-a", Namespace: "platform"},
		Spec: apiv1alpha1.WorkspaceSpec{
			TenantRef: apiv1alpha1.LocalObjectReference{Name: "tenant-a"},
		},
	}
	workspaceB := &apiv1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{Name: "workspace-b", Namespace: "platform"},
		Spec: apiv1alpha1.WorkspaceSpec{
			TenantRef: apiv1alpha1.LocalObjectReference{Name: "tenant-a"},
		},
	}
	workspaceC := &apiv1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{Name: "workspace-c", Namespace: "platform"},
		Spec: apiv1alpha1.WorkspaceSpec{
			TenantRef: apiv1alpha1.LocalObjectReference{Name: "tenant-b"},
		},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.Tenant{}, &apiv1alpha1.Workspace{}).
		WithObjects(tenant, workspaceA, workspaceB, workspaceC).
		Build()
	reconciler := &TenantReconciler{Client: kubeClient, Scheme: scheme}

	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "platform", Name: "tenant-a"}}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	var updated apiv1alpha1.Tenant
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &updated); err != nil {
		t.Fatalf("get Tenant returned error: %v", err)
	}
	if updated.Status.Phase != "Ready" {
		t.Fatalf("expected Ready phase, got %q", updated.Status.Phase)
	}
	if updated.Status.WorkspaceCount != 2 {
		t.Fatalf("expected workspace count 2, got %#v", updated.Status)
	}
	if len(updated.Status.Conditions) != 1 || updated.Status.Conditions[0].Reason != "WorkspaceCountComputed" {
		t.Fatalf("expected WorkspaceCountComputed condition, got %#v", updated.Status.Conditions)
	}
}
