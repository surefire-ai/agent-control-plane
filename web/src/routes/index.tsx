import { createBrowserRouter, Navigate } from "react-router-dom";
import { AppShell } from "@/components/layout/AppShell";
import { TenantListPage } from "@/pages/TenantListPage";
import { WorkspaceListPage } from "@/pages/WorkspaceListPage";
import { WorkspaceDetailPage } from "@/pages/WorkspaceDetailPage";
import { WorkspaceCreatePage } from "@/pages/WorkspaceCreatePage";
import { ProductAreaPage } from "@/pages/ProductAreaPage";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <Navigate to="/tenants" replace /> },
      { path: "tenants", element: <TenantListPage /> },
      { path: "tenants/:tenantId/workspaces", element: <WorkspaceListPage /> },
      { path: "tenants/:tenantId/workspaces/new", element: <WorkspaceCreatePage /> },
      { path: "tenants/:tenantId/workspaces/:workspaceId", element: <WorkspaceDetailPage /> },
      { path: "tenants/:tenantId/agents", element: <ProductAreaPage area="agents" /> },
      { path: "tenants/:tenantId/evaluations", element: <ProductAreaPage area="evaluations" /> },
      { path: "tenants/:tenantId/providers", element: <ProductAreaPage area="providers" /> },
      { path: "tenants/:tenantId/settings", element: <ProductAreaPage area="settings" /> },
    ],
  },
]);
