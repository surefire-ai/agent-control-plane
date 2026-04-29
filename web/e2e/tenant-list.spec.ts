import { test, expect } from "@playwright/test";

test.describe("Tenant List Page", () => {
  test("redirects from / to /tenants", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveURL(/\/tenants/);
  });

  test("displays tenant list", async ({ page }) => {
    await page.goto("/tenants");
    await expect(page.getByRole("heading", { name: "Tenants" })).toBeVisible();
    const table = page.getByRole("table");
    await expect(table.getByText("Demo Tenant")).toBeVisible();
    await expect(table.getByText("Enterprise Tenant")).toBeVisible();
  });

  test("clicking a tenant navigates to its workspaces", async ({ page }) => {
    await page.goto("/tenants");
    await page.getByRole("table").getByText("Demo Tenant").click();
    await expect(page).toHaveURL(/\/tenants\/t_demo\/workspaces/);
    await expect(page.getByRole("heading", { name: "Workspaces" })).toBeVisible();
  });

  test("sidebar tenant switcher shows current tenant", async ({ page }) => {
    await page.goto("/tenants/t_demo/workspaces");
    const switcher = page.locator("#tenant-select");
    await expect(switcher).toHaveValue("t_demo");
  });
});
