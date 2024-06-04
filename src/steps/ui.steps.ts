import { Given, When, Then } from "@cucumber/cucumber";
import { ICustomWorld } from "../support/custom-world";
import { config } from "../support/config";
import { expect } from "@playwright/test";

const slowExpect = expect.configure({ timeout: 10000 });

const lastEventSeen = async function (page: any, event: string) {
  const lastEvent = page.locator("#last-event");
  await slowExpect(lastEvent).toContainText(event);
};

Given(
  "a system in state {string}",
  async function (this: ICustomWorld, state: string) {
    const page = this.page!;
    await page.goto(config.BASE_URL);

    const stateNode = page.locator(`[data-id=${state}]`);
    await slowExpect(stateNode).toHaveClass(/\binProgress\b/);
  },
);

When("the system {string} is requested", async function (command: string) {
  const page = this.page!;
  await page.getByRole("link", { name: command }).click();
});

Then("work is completed", async function () {
  await lastEventSeen(this.page!, "WorkDone");
});

Then("the request is ignored", async function () {
  await lastEventSeen(this.page!, "RequestIgnored");
});

Then("the system is found in state {string}", async function (state: string) {
  const page = this.page!;
  const stateNode = page.locator(`[data-id=${state}]`);
  await slowExpect(stateNode).toHaveClass(/\binProgress\b/);
});

When("some work has progressed", async function () {
  await lastEventSeen(this.page!, "Tick");
});

Then("work is canceled", async function () {
  await lastEventSeen(this.page!, "WorkAborted");
});

Given("two connected clients", async function () {
  const secondPage = this.secondPage!;
  await secondPage.goto(config.BASE_URL);
});

Then("two clients have observed {string}", async function (event: string) {
  await Promise.all([
    lastEventSeen(this.page!, event),
    lastEventSeen(this.secondPage!, event),
  ]);
});
