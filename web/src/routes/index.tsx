import { createBrowserRouter, Navigate } from "react-router-dom";
import { AppShell } from "@/components/layout/AppShell";
import { TenantListPage } from "@/pages/TenantListPage";
import { WorkspaceListPage } from "@/pages/WorkspaceListPage";
import { WorkspaceDetailPage } from "@/pages/WorkspaceDetailPage";
import { WorkspaceCreatePage } from "@/pages/WorkspaceCreatePage";
import { ProductAreaPage } from "@/pages/ProductAreaPage";
import { NotFoundPage } from "@/pages/NotFoundPage";
import { AgentListPage } from "@/pages/AgentListPage";
import { AgentDetailPage } from "@/pages/AgentDetailPage";
import { EvaluationListPage } from "@/pages/EvaluationListPage";
import { ProviderListPage } from "@/pages/ProviderListPage";
import { RunListPage } from "@/pages/RunListPage";

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
      { path: "tenants/:tenantId/agents", element: <AgentListPage /> },
      { path: "tenants/:tenantId/agents/:agentId", element: <AgentDetailPage /> },
      { path: "tenants/:tenantId/evaluations", element: <EvaluationListPage /> },
      { path: "tenants/:tenantId/runs", element: <RunListPage /> },
      { path: "tenants/:tenantId/providers", element: <ProviderListPage /> },
      { path: "tenants/:tenantId/settings", element: <ProductAreaPage area="settings" /> },
      { path: "*", element: <NotFoundPage /> },
    ],
  },
]);
