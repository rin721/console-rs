import { existsSync } from "node:fs";
import { resolve } from "node:path";

const output = resolve("dist/index.html");
if (!existsSync(output)) {
  console.error(`Vite build output is missing: ${output}`);
  process.exit(1);
}
console.log(`Vite build output verified: ${output}`);
