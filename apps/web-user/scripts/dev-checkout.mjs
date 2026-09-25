import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import net from "node:net";
import path from "node:path";

const scriptDirectory = path.dirname(fileURLToPath(import.meta.url));
const appDirectory = path.resolve(scriptDirectory, "..");
const repositoryRoot = path.resolve(appDirectory, "..", "..");
const useShell = process.platform === "win32";
const children = new Set();
let shuttingDown = false;

const mockServices = [
  { name: "Product Discovery Mockoon", port: 4012, file: "oma-product-discovery-mvp.json" },
  { name: "Product Detail Mockoon", port: 4013, file: "oma-product-detail-mvp.json" },
  { name: "Shopping Cart Mockoon", port: 4014, file: "oma-shopping-cart-mvp.json" },
  { name: "Checkout Mockoon", port: 4015, file: "oma-checkout-mvp.json" }
];

function isPortOpen(port) {
  return new Promise((resolve) => {
    const socket = net.createConnection({ host: "127.0.0.1", port });
    const finish = (result) => { socket.destroy(); resolve(result); };
    socket.setTimeout(350);
    socket.once("connect", () => finish(true));
    socket.once("timeout", () => finish(false));
    socket.once("error", () => finish(false));
  });
}

async function waitForPort(port, child) {
  const deadline = Date.now() + 30_000;
  while (Date.now() < deadline) {
    if (await isPortOpen(port)) return;
    if (child.exitCode !== null) throw new Error(`Mockoon port ${port} exited before becoming ready.`);
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  throw new Error(`Timed out waiting for Mockoon port ${port}.`);
}

function start(command, args, cwd, label) {
  const child = spawn(command, args, { cwd, stdio: "inherit", shell: useShell });
  children.add(child);
  child.once("exit", (code) => {
    children.delete(child);
    if (!shuttingDown && code && code !== 0) {
      console.error(`[dev:checkout] ${label} stopped with exit code ${code}.`);
      shutdown(code);
    }
  });
  return child;
}

async function startMock(service) {
  if (await isPortOpen(service.port)) {
    console.log(`[dev:checkout] Reusing ${service.name} on port ${service.port}.`);
    return;
  }
  console.log(`[dev:checkout] Starting ${service.name} on port ${service.port}...`);
  const child = start("npx", ["@mockoon/cli", "start", "--disable-log-to-file", "--data", path.join("mocks", "mockoon", service.file)], repositoryRoot, service.name);
  await waitForPort(service.port, child);
}

function shutdown(exitCode = 0) {
  if (shuttingDown) return;
  shuttingDown = true;
  for (const child of children) {
    if (!child.pid) continue;
    if (process.platform === "win32") spawn("taskkill", ["/pid", String(child.pid), "/T", "/F"], { stdio: "ignore" });
    else child.kill("SIGINT");
  }
  setTimeout(() => process.exit(exitCode), 500);
}

process.once("SIGINT", () => shutdown(0));
process.once("SIGTERM", () => shutdown(0));

try {
  for (const service of mockServices) await startMock(service);
  if (await isPortOpen(3000)) {
    console.log("[dev:checkout] Reusing Next.js on http://localhost:3000.");
    console.log("[dev:checkout] Cart and Checkout are ready. Press Ctrl+C to stop services started by this command.");
    process.stdin.resume();
  } else {
    console.log("[dev:checkout] All mock services are ready. Starting Next.js...");
    const web = start("npm", ["run", "dev"], appDirectory, "Next.js");
    web.once("exit", (code) => shutdown(code ?? 0));
  }
} catch (error) {
  console.error(`[dev:checkout] ${error instanceof Error ? error.message : String(error)}`);
  shutdown(1);
}
