import { test, expect } from "@playwright/test";

test.describe("Agent List Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/tenants/t_demo/agents");
  });

  test("displays tenant agents with runtime and model metadata", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Agents" })).toBeVisible();

    const table = page.getByRole("table");
    await expect(table.getByText("EHS ReAct Agent")).toBeVisible();
    await expect(table.getByText("Evaluation Guard")).toBeVisible();
    await expect(table.getByText("qwen", { exact: true })).toBeVisible();
    await expect(table.getByText("eino", { exact: true })).toHaveCount(2);
  });

  test("does not show agents from another tenant", async ({ page }) => {
    const table = page.getByRole("table");
    await expect(table.getByText("Enterprise Ops Agent")).not.toBeVisible();
  });
});
