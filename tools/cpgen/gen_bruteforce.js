// Generates oracle output for Text Encoding Brute Force by reproducing
// CyberChef's run() (cptable + Utils.byteArrayToUtf8) over all 152 charsets.
const fs = require("fs");
const path = require("path");
const cptable = require("codepage");

const cyberchef = process.argv[2];
const outDir = process.argv[3];
const chrEnc = fs.readFileSync(path.join(cyberchef, "src/core/lib/ChrEnc.mjs"), "utf8");
const body = chrEnc.match(/CHR_ENC_CODE_PAGES\s*=\s*\{([\s\S]*?)\n\}/)[1];
const pairs = [...body.matchAll(/"([^"]+)"\s*:\s*(\d+)/g)].map(m => [m[1], +m[2]]);

function byteArrayToUtf8(bytes) {
  const arr = Uint8Array.from(bytes);
  try { return new TextDecoder("utf-8", { fatal: true }).decode(arr); }
  catch { return Array.from(arr, b => String.fromCharCode(b)).join(""); } // Latin1 fallback
}
function hexToBytes(h) { h = h.replace(/\s/g, ""); const a = []; for (let i = 0; i < h.length; i += 2) a.push(parseInt(h.substr(i, 2), 16)); return a; }

function bruteforce(mode, input) {
  const out = {};
  for (const [name, cp] of pairs) {
    try {
      out[name] = mode === "Decode"
        ? cptable.utils.decode(cp, input)
        : byteArrayToUtf8(cptable.utils.encode(cp, input));
    } catch (e) { out[name] = "Could not decode."; }
  }
  return JSON.stringify(out, null, 4);
}

const decodeHex = "cf f0 e8 e2 e5 f2"; // Windows-1251 bytes for "Привет"
const encodeStr = "Привет héllo 中文!";
const result = {
  decodeHex,
  decodeJson: bruteforce("Decode", hexToBytes(decodeHex)),
  encodeStr,
  encodeJson: bruteforce("Encode", encodeStr),
};
fs.mkdirSync(outDir, { recursive: true });
fs.writeFileSync(path.join(outDir, "textbruteforce_vectors.json"), JSON.stringify(result));
console.log("wrote textbruteforce_vectors.json");
