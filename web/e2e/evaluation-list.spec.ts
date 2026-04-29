import { test, expect } from "@playwright/test";

test.describe("Evaluation List Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/tenants/t_demo/evaluations");
  });

  test("displays tenant evaluations with score and gate metadata", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Evaluations" })).toBeVisible();

    const table = page.getByRole("table");
    await expect(table.getByText("EHS Regression Gate")).toBeVisible();
    await expect(table.getByText("Guardrail Release Check")).toBeVisible();
    await expect(table.getByText("ehs-golden-set")).toBeVisible();
    await expect(table.getByText("94%")).toBeVisible();
    await expect(table.getByText("Failed", { exact: true }).first()).toBeVisible();
  });

  test("does not show evaluations from another tenant", async ({ page }) => {
    const table = page.getByRole("table");
    await expect(table.getByText("Enterprise Ops Weekly")).not.toBeVisible();
  });
});
