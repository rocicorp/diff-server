const util = require('util');
const fs = require('fs');
const program = require('commander');
const getStdin = require('get-stdin');
const sha1 = require('sha1');
const touch = require('touch');
const [readFile, writeFile] = ['readFile', 'writeFile']
    .map(n => util.promisify(fs[n]));

async function opReg() {
    const code = await getStdin();
    const v = parse(code);
    if (typeof v != 'function') {
        console.error('op code must evaluate to a function, not: %s', typeof v);
        return;
    }

    const p = program.opsfile || './ops.json';
    const r = await read(p);
    const h = sha1(code);
    r[h] = code;
    await writeFile(p, JSON.stringify(r, null, 2));
}

async function read() {
    // TODO: sandbox, obvs
    const p = program.opsfile || './ops.json';
    await touch(p);
    const data = await readFile(p, 'utf8');
    if (!data) {
        return {};
    }
    return JSON.parse(data);
}

async function load() {
    const r = await read();
    for (let k in r) {
        r[k] = parse(r[k]);
    }
    return r;
}

function parse(code) {
    return eval("(" + code + ")");
}

async function list() {
    const ops = await read();
    for (let [k, v] of Object.entries(ops)) {
        console.log(`${k}\n${v}`);
    }
}

async function getOp(nameOrHash) {
    const ops = await load();
    if (ops[nameOrHash]) {
        return {hash: nameOrHash, op: ops[nameOrHash]};
    }
    let h = null;
    for (k in ops) {
        if (ops[k].name == nameOrHash) {
            if (h) {
                throw new Error('Multiple definitions for ' + nameOrHash);
            }
            h = k;
        }
    }
    return h && {hash: h, op: ops[h]};
}

module.exports = {opReg, list, getOp};
