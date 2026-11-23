// mise run size: the build's size, a report and never a gate. It reads every file under dist/ and
// prints three totals: the raw bytes, and the bytes each file compresses to, one at a time as a
// server sends it, under gzip and under brotli at their highest levels, with node:zlib. No budget
// is enforced; the totals are for reading, a build's against the last.
//
//   node scripts/size.ts [dir]

import { readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { brotliCompressSync, constants, gzipSync } from "node:zlib";

/** A build's totals: its file count, and its bytes raw, gzipped, and brotli-compressed. */
export interface Totals {
  files: number;
  raw: number;
  gzip: number;
  brotli: number;
}

/** Returns the totals of every file under dir, its subdirectories' included. */
export function measure(dir: string): Totals {
  const totals: Totals = { files: 0, raw: 0, gzip: 0, brotli: 0 };
  for (const entry of readdirSync(dir, { recursive: true, withFileTypes: true })) {
    if (!entry.isFile()) continue;
    const bytes = readFileSync(join(entry.parentPath, entry.name));
    totals.files += 1;
    totals.raw += bytes.length;
    totals.gzip += gzipSync(bytes, { level: constants.Z_BEST_COMPRESSION }).length;
    totals.brotli += brotliCompressSync(bytes, {
      params: {
        [constants.BROTLI_PARAM_QUALITY]: constants.BROTLI_MAX_QUALITY,
        [constants.BROTLI_PARAM_SIZE_HINT]: bytes.length,
      },
    }).length;
  }
  return totals;
}

/** Returns the report of totals for dir: a heading line, then one line per measure. */
export function report(dir: string, totals: Totals): string {
  const width = Math.max(...[totals.raw, totals.gzip, totals.brotli].map((n) => format(n).length));
  const line = (label: string, n: number) => `  ${label.padEnd(7)}${format(n).padStart(width)} B`;
  return [
    `size: ${dir} (${totals.files} files)`,
    line("raw", totals.raw),
    line("gzip", totals.gzip),
    line("brotli", totals.brotli),
  ].join("\n");
}

const format = (n: number): string => n.toLocaleString("en-US");

if (import.meta.main) {
  const dir = process.argv[2] ?? "dist";
  console.log(report(dir, measure(dir)));
}
