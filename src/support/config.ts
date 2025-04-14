import { LaunchOptions } from "@playwright/test";

const browserOptions: LaunchOptions = {
  slowMo: 0,
  args: [
    "--use-fake-ui-for-media-stream",
    "--use-fake-device-for-media-stream",
  ],
  firefoxUserPrefs: {
    "media.navigator.streams.fake": true,
    "media.navigator.permission.disabled": true,
  },
};

export var config = {
  browser: process.env.BROWSER ?? "chromium",
  browserOptions,
  startServer: process.env.SUT_START_SERVER === "true",
  BASE_URL: process.env.SUT_BASE_URL ?? "http://localhost:8080",
  IMG_THRESHOLD: { threshold: 0.4 },
  BASE_API_URL: "http://localhost:8080",
  server: null as any,
};
