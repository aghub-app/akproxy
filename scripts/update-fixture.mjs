// Local GitHub Releases fixture for the native updater smoke test.
// Usage: node scripts/update-fixture.mjs <asset-directory> <version> [port] [bad-checksum]
import { createServer } from "node:http";
import { createReadStream, statSync } from "node:fs";
import { createHash } from "node:crypto";
import { resolve, join } from "node:path";

const [directory, version, port = "18765", fault] = process.argv.slice(2);
if (!directory || !/^\d+\.\d+\.\d+$/.test(version)) throw new Error("Pass an asset directory and MAJOR.MINOR.PATCH version");
const asset = "akproxy-darwin-universal.zip";
const file = join(resolve(directory), asset);
const hash = createHash("sha256");
for await (const chunk of createReadStream(file)) hash.update(chunk);
const checksum = fault === "bad-checksum" ? "0".repeat(64) : hash.digest("hex");
const origin = `http://127.0.0.1:${port}`;
createServer((request, response) => {
  console.log(request.method, request.url);
  if (request.url === "/repos/aghub-app/akproxy/releases/latest") {
    response.setHeader("Content-Type", "application/json");
    response.end(JSON.stringify({ tag_name: `v${version}`, name: `akproxy ${version}`, body: "Local update smoke test", draft: false, prerelease: false,
      assets: [{ name: asset, size: statSync(file).size, browser_download_url: `${origin}/${asset}` },
        { name: "SHA256SUMS", browser_download_url: `${origin}/SHA256SUMS` }] }));
  } else if (request.url === "/SHA256SUMS") {
    response.end(`${checksum}  ${asset}\n`);
  } else if (request.url === `/${asset}`) {
    response.setHeader("Content-Length", statSync(file).size);
    createReadStream(file).pipe(response);
  } else {
    response.writeHead(404).end();
  }
}).listen(Number(port), "127.0.0.1", () => console.log(`Fixture listening on ${origin}`));
