import { readFile, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { spawnSync } from "node:child_process";

const root = process.cwd();
const nextEnvPath = join(root, "next-env.d.ts");
const originalNextEnv = await readFile(nextEnvPath, "utf8");

function run(name, entry, args) {
  const result = spawnSync(process.execPath, [join(root, entry), ...args], { cwd: root, stdio: "inherit" });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`${name} ${args.join(" ")} exited with status ${result.status ?? "unknown"}.`);
}

try {
  run("next", "node_modules/next/dist/bin/next", ["typegen"]);
  run("tsc", "node_modules/typescript/bin/tsc", ["--noEmit"]);
} finally {
  await writeFile(nextEnvPath, originalNextEnv, "utf8");
}
