import { spawn } from "node:child_process";

const mode = process.argv[2] ?? "auto";
if (!["mock", "real", "auto"].includes(mode)) {
  console.error("Usage: npm run preview:data -- mock|real|auto");
  process.exit(1);
}
console.log("Storefront data source:", mode);
const child = spawn(process.execPath, ["node_modules/next/dist/bin/next", "start", "-H", "127.0.0.1", "-p", process.env.PORT ?? "4321"], {
  env: { ...process.env, STOREFRONT_DATA_MODE: mode, NEXT_PUBLIC_RECOMMENDATIONS_ENABLED: "true" },
  stdio: "inherit",
  windowsHide: true
});
child.on("error", error => { console.error(error.message); process.exitCode = 1; });
child.on("exit", code => { process.exitCode = code ?? 1; });
process.on("SIGINT", () => child.kill("SIGINT"));
process.on("SIGTERM", () => child.kill("SIGTERM"));
