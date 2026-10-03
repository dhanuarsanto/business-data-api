module.exports = {
  apps: [
    {
      name: "message-data-api",
      script: process.platform === "win32" ? "./api.exe" : "./api",
      interpreter: "none",
      cwd: __dirname + "/bin",
      instances: 1,
      kill_timeout: 15000,
      max_memory_restart: "400M",
    },
  ],
};
