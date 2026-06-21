import { existsSync } from "node:fs";
import { resolve } from "node:path";

const indexPath = resolve("build/client/index.html");

if (!existsSync(indexPath)) {
  console.error(`React Router client build output is missing: ${indexPath}`);
  process.exit(1);
}

console.log(`React Router client build output verified: ${indexPath}`);
