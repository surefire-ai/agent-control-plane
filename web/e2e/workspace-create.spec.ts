import { test, expect } from "@playwright/test";
import { cleanupWorkspace } from "./fixtures/api";

const TEST_WS_ID = "ws_e2e_test";

test.describe("Workspace Create Page", () => {
  test.describe.configure({ mode: "serial" });

  test.afterEach(async () => {
    await cleanupWorkspace(TEST_WS_ID);
  });

  test("can create a new workspace", async ({ page }) => {
    await page.goto("/tenants/t_demo/workspaces/new");

    await expect(page.getByRole("heading", { name: "Create Workspace" })).toBeVisible();

    await page.getByLabel("ID").fill(TEST_WS_ID);
    await page.getByLabel("Slug").fill("e2e-test");
    await page.getByLabel("Display Name").fill("E2E Test Workspace");
    await page.getByLabel("Description").fill("Created by Playwright");

    await page.getByRole("button", { name: "Create Workspace" }).click();

    await expect(page).toHaveURL(/\/tenants\/t_demo\/workspaces$/);
    await expect(page.getByRole("table").getByText("E2E Test Workspace")).toBeVisible();
  });

  test("cancel button returns to workspace list", async ({ page }) => {
    await page.goto("/tenants/t_demo/workspaces/new");
    await page.getByRole("button", { name: "Cancel" }).click();
    await expect(page).toHaveURL(/\/tenants\/t_demo\/workspaces$/);
  });
});
