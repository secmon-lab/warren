import { type Page, type Locator } from "@playwright/test";
import { BasePage } from "./BasePage";

export class KnowledgeListPage extends BasePage {
  readonly heading: Locator;
  readonly newKnowledgeButton: Locator;
  readonly categoryFactButton: Locator;
  readonly categoryTechniqueButton: Locator;
  readonly keywordInput: Locator;
  readonly noKnowledgeMessage: Locator;

  constructor(page: Page) {
    super(page);
    this.heading = page.getByRole("heading", { name: "Knowledge Base" });
    this.newKnowledgeButton = page.getByRole("link", { name: "New Knowledge" });
    this.categoryFactButton = page.getByRole("button", { name: "fact" });
    this.categoryTechniqueButton = page.getByRole("button", { name: "technique" });
    this.keywordInput = page.getByPlaceholder("Search by keyword...");
    this.noKnowledgeMessage = page.getByText("No knowledge found.");
  }

  async goto(category: "fact" | "technique" = "fact") {
    await super.goto(`/knowledge/${category}`);
  }

  get tagButtons() {
    return this.page.locator("button.rounded-full");
  }

  get knowledgeItems() {
    return this.page.locator(".border.rounded-md.divide-y > a");
  }

  async selectCategory(category: "fact" | "technique") {
    if (category === "fact") {
      await this.categoryFactButton.click();
    } else {
      await this.categoryTechniqueButton.click();
    }
  }

  async searchByKeyword(keyword: string) {
    await this.keywordInput.fill(keyword);
  }

  async clickTag(tagName: string) {
    await this.page.getByRole("button", { name: tagName, exact: true }).click();
  }
}
