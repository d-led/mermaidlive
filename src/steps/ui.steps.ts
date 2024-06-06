import { Given, When, Then, BeforeAll, AfterAll } from "@cucumber/cucumber";
import { ICustomWorld } from "../support/custom-world";
import { config } from "../support/config";
import { expect } from "@playwright/test";
import { spawn } from "child_process";

const slowExpect = expect.configure({ timeout: 10000 });

const lastEventSeen = async function (page: any, event: string) {
  const lastEvent = page.locator("#last-event");
  await slowExpect(lastEvent).toContainText(event);
};

BeforeAll(async () => {
  if (config.startServer && !config.server) {
    console.log(`Starting server: `);
    var server = spawn("./mermaidlive", [], {
      shell: false,
      stdio: "inherit",
      env: process.env,
    });
    server.on("close", (code, signal) => {
      console.log(
        `child process terminated with ${code} due to receipt of signal ${signal}`,
      );
    });
    config.server = server;

    await waitForUrl(config.BASE_API_URL);
  }
  console.log(`SUT_BASE_URL: ${config.BASE_URL}`);
});

AfterAll(() => {
  if (config.server) {
    console.log("Stopping the server");
    (config.server as any)?.kill("SIGHUP");
  }
});

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

async function waitForUrl(url: string) {
  for (let i = 0; i < 15; i++) {
    console.log(`trying to reach: ${url}`);
    try {
      var res = await fetch(url);
      if (res?.ok) {
        return;
      }
    } catch (e) {
      console.log(`could not reach ${url}: ${e}`);
    }
    await sleep(1);
  }
  console.log(`giving up waiting for ${url}`);
}

async function sleep(seconds: number) {
  await new Promise((resolve) => setTimeout(resolve, seconds * 1000 /*ms*/));
}
