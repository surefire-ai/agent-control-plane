import { test, expect } from "@playwright/test";

test.describe("Workspace List Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/tenants/t_demo/workspaces");
  });

  test("displays workspaces for selected tenant", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Workspaces" })).toBeVisible();
    await expect(page.getByText("Demo Workspace")).toBeVisible();
    await expect(page.getByText("Staging Workspace")).toBeVisible();
  });

  test("shows empty state for tenant with no workspaces", async ({ page }) => {
    await page.goto("/tenants/t_inactive/workspaces");
    await expect(page.getByText("No workspaces")).toBeVisible();
  });

  test("has a button to create new workspace", async ({ page }) => {
    const createButton = page.getByRole("link", { name: "New Workspace" });
    await expect(createButton).toBeVisible();
    await createButton.click();
    await expect(page).toHaveURL(/\/workspaces\/new/);
  });

  test("clicking workspace navigates to detail", async ({ page }) => {
    await page.getByText("Demo Workspace").click();
    await expect(page).toHaveURL(/\/workspaces\/ws_demo/);
  });
});
