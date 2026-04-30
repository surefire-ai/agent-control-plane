import { test, expect } from "@playwright/test";

test.describe("Run list", () => {
  test("shows tenant-scoped agent runs", async ({ page }) => {
    await page.goto("/tenants/t_demo/runs");

    await expect(page.getByRole("heading", { name: "Runs" })).toBeVisible();
    await expect(page.getByRole("cell", { name: "run_ehs_20260429_001" })).toBeVisible();
    await expect(page.getByRole("cell", { name: "run_guardrail_20260429_002" })).toBeVisible();
    await expect(page.getByText("inspection complete")).toBeVisible();
    await expect(page.getByText("release gate failed")).toBeVisible();
  });

  test("does not show runs from another tenant", async ({ page }) => {
    await page.goto("/tenants/t_demo/runs");

    const table = page.getByRole("table");
    await expect(table.getByText("run_enterprise_20260429_003")).not.toBeVisible();
  });
});
