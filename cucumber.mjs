const getWorldParams = () => {
  const params = {};

  return params;
};

const config = {
  requireModule: ["ts-node/register"],
  require: ["src/**/*.ts"],
  format: [
    // 'message:e2e/reports/cucumber-report.ndjson',
    "json:reports/cucumber-report.json",
    //   'html:reports/report.html',
    "summary",
    "progress-bar",
  ],
  formatOptions: { snippetInterface: "async-await" },
  worldParameters: getWorldParams(),
};

config.format.push("@cucumber/pretty-formatter");
export default config;
