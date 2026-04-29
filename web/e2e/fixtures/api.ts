const API = "http://localhost:8090/api/v1";

async function request(path: string, options?: RequestInit) {
  const res = await fetch(`${API}${path}`, {
    headers: { "Content-Type": "application/json", ...options?.headers },
    ...options,
  });
  if (!res.ok) {
    throw new Error(`API ${options?.method ?? "GET"} ${path} failed: ${res.status}`);
  }
  if (res.status === 204) return null;
  return res.json();
}

export const api = {
  get: (path: string) => request(path),
  post: (path: string, body: unknown) =>
    request(path, { method: "POST", body: JSON.stringify(body) }),
  delete: (path: string) => request(path, { method: "DELETE" }),
};

export async function seedWorkspace(data: {
  id: string;
  tenantId: string;
  slug: string;
  displayName: string;
  status?: string;
}) {
  return api.post("/workspaces/", {
    ...data,
    status: data.status ?? "active",
  });
}

export async function cleanupWorkspace(id: string) {
  try {
    await api.delete(`/workspaces/${id}`);
  } catch {
    // ignore cleanup errors
  }
}
