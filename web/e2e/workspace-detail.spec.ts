import { test, expect } from "@playwright/test";

test.describe("Workspace Detail Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/tenants/t_demo/workspaces/ws_demo");
  });

  test("displays workspace details", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Demo Workspace" })).toBeVisible();
    await expect(page.getByText("ws_demo")).toBeVisible();
    await expect(page.getByText("Active", { exact: true })).toBeVisible();
  });

  test("can enter edit mode and cancel", async ({ page }) => {
    await page.getByRole("button", { name: "Edit" }).click();

    await expect(page.getByRole("button", { name: "Save Changes" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Cancel" })).toBeVisible();

    await page.getByRole("button", { name: "Cancel" }).click();
    await expect(page.getByRole("button", { name: "Edit" })).toBeVisible();
  });

  test("can open and close delete dialog", async ({ page }) => {
    await page.getByRole("button", { name: "Delete" }).click();
    await expect(page.getByText("Are you sure")).toBeVisible();

    await page.getByRole("button", { name: "Cancel" }).click();
    await expect(page.getByText("Are you sure")).not.toBeVisible();
  });

  test("shows breadcrumb navigation", async ({ page }) => {
    const breadcrumb = page.getByLabel("Breadcrumb");
    await expect(breadcrumb.getByText("Tenants")).toBeVisible();
    await expect(breadcrumb.getByText("Demo Tenant")).toBeVisible();
    await expect(breadcrumb.getByText("Demo Workspace")).toBeVisible();
  });
});
