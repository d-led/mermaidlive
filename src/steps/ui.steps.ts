import { Given, When , Then} from "@cucumber/cucumber";
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
        await expect.configure({ timeout: 10000 })(waitingState).toHaveClass(/\binProgress\b/);
        break;
      default:
        throw "unknown state: " + state;
    }
  },
);

When('the system {string} is requested', async function (command: string) {
    const page = this.page!;
    await page.getByRole('link', { name: command }).click();
});

Then('work is completed', async function () {
  // Write code here that turns the phrase above into concrete actions
  return 'pending';
});

Then('the request is ignored', async function () {
  // Write code here that turns the phrase above into concrete actions
  return 'pending';
});

Then('the system is found in state {string}', async function (_state: string) {
  // Write code here that turns the phrase above into concrete actions
  return 'pending';
});

When('some work has progressed', async function () {
  // Write code here that turns the phrase above into concrete actions
  return 'pending';
});

Then('work is canceled', async function () {
  // Write code here that turns the phrase above into concrete actions
  return 'pending';
});

Given('two connected clients', async function () {
  // Write code here that turns the phrase above into concrete actions
  return 'pending';
});

Then('two clients have observed {string}', async function (_event: string) {
  // Write code here that turns the phrase above into concrete actions
  return 'pending';
});
