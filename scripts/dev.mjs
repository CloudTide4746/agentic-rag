import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";

const root = new URL("../", import.meta.url);
const windows = process.platform === "win32";
const children = new Set();
let stopping = false;

// Stop only processes started by this runner, including go run's API child.
async function stop(code) {
  if (stopping) return;
  stopping = true;
  process.exitCode = code;
  await Promise.all([...children].map(child => new Promise(resolve => {
    if (!child.pid) return resolve();
    if (windows) {
      const killer = spawn("taskkill", ["/pid", String(child.pid), "/t", "/f"], {
        stdio: "ignore", windowsHide: true,
      });
      killer.once("error", () => { child.kill(); resolve(); });
      killer.once("exit", resolve);
    } else {
      try {
        process.kill(-child.pid, "SIGTERM");
      } catch (error) {
        if (error.code !== "ESRCH") {
          console.error("Failed to stop development process:", error.message);
          process.exitCode = 1;
        }
      }
      resolve();
    }
  })));
}

function start(name, command, args, directory) {
  const child = spawn(command, args, {
    cwd: fileURLToPath(new URL(directory, root)),
    stdio: "inherit", detached: !windows, windowsHide: true,
  });
  children.add(child);
  child.once("error", error => {
    console.error(`${name} could not start: ${error.message}`);
    void stop(1);
  });
  child.once("exit", (code, signal) => {
    if (!windows) children.delete(child);
    if (!stopping) {
      console.error(`${name} exited (${signal ?? code}); stopping development services.`);
      void stop(code || 1);
    }
  });
}

process.once("SIGINT", () => { void stop(0); });
process.once("SIGTERM", () => { void stop(0); });

start("Go API", "go", ["run", "./cmd/api", "-config", "config.yaml"], "apps/api/");
start("Vite", process.execPath, ["node_modules/vite/bin/vite.js", "--host", "127.0.0.1"], "apps/web/");
