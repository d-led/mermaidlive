import { Given } from "@cucumber/cucumber";
import { ICustomWorld } from "../support/custom-world";
import { config } from "../support/config";
import { expect } from "@playwright/test";

Given(
  "a system in state {string}",
  async function (this: ICustomWorld, state: string) {
    const page = this.page!;
    await page.goto(config.BASE_URL);

    switch (state) {
      case "waiting":
        const waitingState = page.locator("[data-id=waiting]");
        await expect(waitingState).toHaveClass(/\binProgress\b/);
        break;
      default:
        throw "unknown state: " + state;
    }
  },
);
