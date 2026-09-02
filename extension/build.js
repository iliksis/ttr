// Bundles the background and content scripts into classic (non-module)
// scripts under dist/, alongside a copy of manifest.json — the folder to
// load unpacked in Chrome and Firefox.
import { build } from "esbuild";
import { cpSync, mkdirSync } from "node:fs";

mkdirSync("dist", { recursive: true });

await build({
  entryPoints: ["src/background.js", "src/content.js"],
  bundle: true,
  format: "iife",
  target: "es2020",
  outdir: "dist",
});

cpSync("manifest.json", "dist/manifest.json");

console.log("Built extension into dist/ — load that folder unpacked.");
