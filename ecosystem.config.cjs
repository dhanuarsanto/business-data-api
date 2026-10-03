const fs = require("fs");
const path = require("path");

const version = fs.readFileSync(path.join(__dirname, "version.txt"), "utf8").trim();

module.exports = {
  apps: [
    {
      name: "message-data-api",
      script: process.platform === "win32" ? "./bin/api.exe" : "./bin/api",
      interpreter: "none",
      cwd: __dirname + "/bin",
      instances: 1,
      kill_timeout: 15000,
      max_memory_restart: "400M",
      version: version,
    },
  ],
};