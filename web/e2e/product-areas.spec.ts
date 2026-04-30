import { test, expect } from "@playwright/test";

const areas = [
  { label: "Agents", path: "/agents", heading: "Agents" },
  { label: "Evaluations", path: "/evaluations", heading: "Evaluations" },
  { label: "Runs", path: "/runs", heading: "Runs" },
  { label: "Providers", path: "/providers", heading: "Providers" },
  { label: "Settings", path: "/settings", heading: "Settings" },
];

test.describe("Product area navigation", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/tenants/t_demo/workspaces");
  });

  for (const area of areas) {
    test(`opens ${area.label}`, async ({ page }) => {
      await page.getByRole("link", { name: area.label }).click();
      await expect(page).toHaveURL(new RegExp(`/tenants/t_demo${area.path}$`));
      await expect(page.getByRole("heading", { name: area.heading })).toBeVisible();
      await expect(page.getByLabel("Breadcrumb").getByText(area.heading)).toBeVisible();
    });
  }
});
