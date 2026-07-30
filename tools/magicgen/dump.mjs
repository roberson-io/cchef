/**
 * Extracts the tables the Magic operation needs from CyberChef into
 * magicdata.json, for tools/magicgen/gen.go to turn into Go.
 *
 * Three things are pulled out:
 *   - the detection checks every operation declares in OperationConfig.json,
 *   - the byte-frequency profiles Magic compares data against to guess a
 *     language, in both the common and the extensive set,
 *   - the language code to name mapping used to report a guess.
 *
 * Run from the repository root with CyberChef checked out alongside:
 *
 *     node tools/magicgen/dump.mjs ../CyberChef > tools/magicgen/magicdata.json
 */
import {readFileSync} from "fs";
import {resolve} from "path";

// Resolved against the working directory, not this script, so the path given on
// the command line means what it looks like it means.
const root = resolve(process.argv[2] || "../CyberChef");
const config = JSON.parse(
    readFileSync(`${root}/src/core/config/OperationConfig.json`, "utf8"));

/** Every check an operation declares, flattened into one list. */
function checks() {
    const out = [];
    for (const [op, meta] of Object.entries(config)) {
        if (!("checks" in meta)) continue;
        for (const check of meta.checks) {
            out.push({
                op,
                pattern: check.pattern ?? "",
                flags: check.flags ?? "",
                args: check.args ?? [],
                useful: !!check.useful,
                entropyRange: check.entropyRange ?? null,
                output: check.output
                    ? {
                        pattern: check.output.pattern ?? "",
                        flags: check.output.flags ?? "",
                        entropyRange: check.output.entropyRange ?? null,
                        mime: check.output.mime ?? "",
                    }
                    : null,
            });
        }
    }
    return out;
}

// The frequency tables and the language names are plain literals in the source
// rather than exports, so they are read out of the module text.
const source = readFileSync(`${root}/src/core/lib/Magic.mjs`, "utf8");

/** Evaluates one object literal assigned to a named const in the source. */
function objectLiteral(name) {
    const at = source.indexOf(`const ${name} = `);
    if (at < 0) throw new Error(`${name} not found`);
    const open = source.indexOf("{", at);
    let depth = 0;
    for (let i = open; i < source.length; i++) {
        if (source[i] === "{") depth++;
        else if (source[i] === "}") {
            depth--;
            if (depth === 0) {
                // eslint-disable-next-line no-eval
                return eval(`(${source.slice(open, i + 1)})`);
            }
        }
    }
    throw new Error(`${name} is not closed`);
}

const common = objectLiteral("COMMON_LANG_FREQS");
// The extensive set is written as the common set plus more, so build it the
// same way rather than trying to read the merged result.
const extensiveExtra = (() => {
    const at = source.indexOf("const EXTENSIVE_LANG_FREQS = Object.assign({}, COMMON_LANG_FREQS,");
    if (at < 0) throw new Error("EXTENSIVE_LANG_FREQS not found");
    const open = source.indexOf("{", source.indexOf(",", at));
    let depth = 0;
    for (let i = open; i < source.length; i++) {
        if (source[i] === "{") depth++;
        else if (source[i] === "}") {
            depth--;
            // eslint-disable-next-line no-eval
            if (depth === 0) return eval(`(${source.slice(open, i + 1)})`);
        }
    }
    throw new Error("EXTENSIVE_LANG_FREQS is not closed");
})();
const extensive = Object.assign({}, common, extensiveExtra);

/** The language code to name mapping, read out of codeToLanguage. */
function languageNames() {
    const at = source.indexOf("static codeToLanguage(code)");
    if (at < 0) throw new Error("codeToLanguage not found");
    const open = source.indexOf("{", source.indexOf("return", at));
    let depth = 0;
    for (let i = open; i < source.length; i++) {
        if (source[i] === "{") depth++;
        else if (source[i] === "}") {
            depth--;
            // eslint-disable-next-line no-eval
            if (depth === 0) return eval(`(${source.slice(open, i + 1)})`);
        }
    }
    throw new Error("codeToLanguage is not closed");
}

process.stdout.write(JSON.stringify({
    checks: checks(),
    commonLangs: common,
    extensiveLangs: extensive,
    languageNames: languageNames(),
}, null, 1));
