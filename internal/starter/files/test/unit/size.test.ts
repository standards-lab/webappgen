// The size report's totals, over a scratch build in TMPDIR: every file under the directory counted,
// its subdirectories' included, raw and each compressed one file at a time.

import assert from "node:assert/strict";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { after, before, test } from "node:test";
import { brotliCompressSync, constants, gzipSync } from "node:zlib";

import { measure, report } from "../../scripts/size.ts";

let dir: string;
const index = "<!doctype html>\n".repeat(64);
const chunk = "export const x = 1;\n".repeat(256);

before(() => {
  dir = mkdtempSync(join(tmpdir(), "size-"));
  mkdirSync(join(dir, "assets"));
  writeFileSync(join(dir, "index.html"), index);
  writeFileSync(join(dir, "assets", "index.js"), chunk);
});
after(() => rmSync(dir, { recursive: true, force: true }));

test("measure counts every file, raw, in the directory and below", () => {
  const totals = measure(dir);
  assert.equal(totals.files, 2);
  assert.equal(totals.raw, index.length + chunk.length);
});

test("measure sums each file compressed apart, at the highest gzip and brotli levels", () => {
  const files = [index, chunk].map((text) => Buffer.from(text));
  const sum = (compress: (bytes: Buffer) => Buffer) =>
    files.reduce((total, bytes) => total + compress(bytes).length, 0);
  const totals = measure(dir);
  assert.equal(
    totals.gzip,
    sum((bytes) => gzipSync(bytes, { level: constants.Z_BEST_COMPRESSION })),
  );
  assert.equal(
    totals.brotli,
    sum((bytes) =>
      brotliCompressSync(bytes, {
        params: {
          [constants.BROTLI_PARAM_QUALITY]: constants.BROTLI_MAX_QUALITY,
          [constants.BROTLI_PARAM_SIZE_HINT]: bytes.length,
        },
      }),
    ),
  );
  assert.ok(totals.gzip < totals.raw && totals.brotli < totals.raw);
});

test("measure of an empty directory is all zeros", () => {
  const empty = mkdtempSync(join(tmpdir(), "size-empty-"));
  try {
    assert.deepEqual(measure(empty), { files: 0, raw: 0, gzip: 0, brotli: 0 });
  } finally {
    rmSync(empty, { recursive: true, force: true });
  }
});

test("report prints the directory, the file count, and each total, aligned", () => {
  assert.equal(
    report("dist", { files: 3, raw: 12345, gzip: 4567, brotli: 890 }),
    ["size: dist (3 files)", "  raw    12,345 B", "  gzip    4,567 B", "  brotli    890 B"].join(
      "\n",
    ),
  );
});
