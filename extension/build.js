// Bundles the background and content scripts into classic (non-module)
// scripts under dist/, alongside a copy of manifest.json — the folder to
// load unpacked in Chrome and Firefox.
import { build } from "esbuild";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { SERVER_URL } from "./src/config.js";

mkdirSync("dist", { recursive: true });

await build({
  entryPoints: ["src/background.js", "src/content.js"],
  bundle: true,
  format: "iife",
  target: "es2020",
  outdir: "dist",
});

// The Server's host_permissions entry is derived from config.js's
// SERVER_URL here, rather than duplicated as a literal in manifest.json, so
// the two can't drift out of sync.
const manifest = JSON.parse(readFileSync("manifest.json", "utf8"));
manifest.host_permissions.push(`${new URL(SERVER_URL).origin}/*`);
writeFileSync("dist/manifest.json", JSON.stringify(manifest, null, 2));

console.log("Built extension into dist/ — load that folder unpacked.");
