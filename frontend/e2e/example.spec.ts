import { test, expect } from "./fixtures/base";

test.describe("Homepage", () => {
  test("should load the homepage", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator("h1")).toBeVisible();
    await expect(page.locator("header a").first()).toBeVisible();
  });

  test("should navigate to 404 page for unknown routes", async ({ page }) => {
    await page.goto("/unknown-route");
    await expect(page.getByText(/page not found/i)).toBeVisible();
  });
});
