import { test, expect } from "../fixtures";
import { KnowledgeListPage } from "../pages/KnowledgePage";

test.describe("Knowledge", () => {
  test("should display knowledge list page", async ({
    authenticatedPage: page,
  }) => {
    const knowledgePage = new KnowledgeListPage(page);
    await knowledgePage.goto();
    await expect(knowledgePage.heading).toBeVisible();
    await expect(knowledgePage.newKnowledgeButton).toBeVisible();
  });

  test("should display category filter buttons", async ({
    authenticatedPage: page,
  }) => {
    const knowledgePage = new KnowledgeListPage(page);
    await knowledgePage.goto();
    await expect(knowledgePage.categoryFactButton).toBeVisible();
    await expect(knowledgePage.categoryTechniqueButton).toBeVisible();
  });

  test("should switch category filter and update URL", async ({
    authenticatedPage: page,
  }) => {
    const knowledgePage = new KnowledgeListPage(page);
    await knowledgePage.goto();

    // fact is selected by default
    await expect(page).toHaveURL(/\/knowledge\/fact$/);
    await expect(knowledgePage.categoryFactButton).toHaveClass(/bg-blue-600/);
    await expect(knowledgePage.categoryTechniqueButton).not.toHaveClass(/bg-blue-600/);

    // Switch to technique
    await knowledgePage.selectCategory("technique");
    await expect(page).toHaveURL(/\/knowledge\/technique$/);
    await expect(knowledgePage.categoryTechniqueButton).toHaveClass(/bg-blue-600/);
    await expect(knowledgePage.categoryFactButton).not.toHaveClass(/bg-blue-600/);
  });

  test("should display keyword search input", async ({
    authenticatedPage: page,
  }) => {
    const knowledgePage = new KnowledgeListPage(page);
    await knowledgePage.goto();
    await expect(knowledgePage.keywordInput).toBeVisible();
  });

  test("should show new knowledge button that links to create page", async ({
    authenticatedPage: page,
  }) => {
    const knowledgePage = new KnowledgeListPage(page);
    await knowledgePage.goto();
    await expect(knowledgePage.newKnowledgeButton).toHaveAttribute("href", "/knowledge/new");
  });

  test("should navigate to knowledge page from sidebar", async ({
    authenticatedPage: page,
  }) => {
    const knowledgePage = new KnowledgeListPage(page);
    await knowledgePage.navigateTo("knowledge");
    await expect(page).toHaveURL(/\/knowledge\/fact$/);
    await expect(knowledgePage.heading).toBeVisible();
  });

  test("should navigate to new knowledge page", async ({
    authenticatedPage: page,
  }) => {
    const knowledgePage = new KnowledgeListPage(page);
    await knowledgePage.goto();
    await knowledgePage.newKnowledgeButton.click();
    await expect(page).toHaveURL(/\/knowledge\/new$/);
    await expect(page.getByRole("heading", { name: "New Knowledge" })).toBeVisible();
  });

  test("should display create knowledge form fields", async ({
    authenticatedPage: page,
  }) => {
    await page.goto("/knowledge/new");

    // Category selector
    await expect(page.getByRole("button", { name: "fact" })).toBeVisible();
    await expect(page.getByRole("button", { name: "technique" })).toBeVisible();

    // Form fields
    await expect(page.getByPlaceholder(/svchost.exe/)).toBeVisible();
    await expect(page.getByPlaceholder(/Write facts or techniques/)).toBeVisible();
    await expect(page.getByPlaceholder("Reason for this change...")).toBeVisible();

    // Action buttons
    await expect(page.getByRole("button", { name: "Save" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Cancel" })).toBeVisible();
  });
});
