import { test, expect } from "@playwright/test";

test.describe("Provider list", () => {
  test("shows tenant-scoped provider accounts", async ({ page }) => {
    await page.goto("/tenants/t_demo/providers");

    await expect(page.getByRole("heading", { name: "Providers" })).toBeVisible();
    await expect(page.getByRole("cell", { name: "Qwen Production" })).toBeVisible();
    await expect(page.getByRole("cell", { name: "DeepSeek Release Gate" })).toBeVisible();
    await expect(page.getByText("secret://demo/qwen-api-key")).toBeVisible();
    await expect(page.getByText("Domestic").first()).toBeVisible();
  });
});
