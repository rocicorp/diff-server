const fs = require('fs').promises;
const tmp = require('tmp-promise');
const util = require('util');
const touch = require('touch');
const { exec, spawn } = require('child_process');
const pexec = util.promisify(exec);
const ops = require('./ops');

const LOCAL_BRANCH = 'local';
const REMOTE_BRANCH = 'remote';

class Database {
    constructor(path, branch) {
        this.path_ = path;
        this.branch_ = branch;
        this.root_ = null;
    }

    async get() {
        if (!await hasBranch(this.path_, this.branch_)) {
            return {};
        }
        return JSON.parse(await noms('json', 'out', `${this.path_}::${this.branch_}.value`, '@'));
    }

    set(root) {
        this.root_ = root;
    }
}

async function opCmd(dbName, opName, args) {
    await runOp(dbName, LOCAL_BRANCH, opName, args);
}

async function push(dbPath, logPath) {
    const local = await getLog(dbPath, LOCAL_BRANCH);
    await touch(logPath);
    const remote = (await fs.readFile(logPath, {encoding: 'utf8', flag: 'r'})).split('\n');
    let i = remote.findIndex((v, i) => v.split(' ')[0] != local[i]);
    const f = await fs.open(logPath, 'a');
    for (let l; l = local[i]; i++) {
        const [name, args] = await getOpFromCommit(dbPath, l);
        await f.writeFile([l, name, JSON.stringify(args)].join(' ') + '\n');
    }
    await f.close();
}

async function pull(dbPath, logPath) {
    // find place where remote branch and log diverge
    const remote = await getLog(dbPath, REMOTE_BRANCH);
    await touch(logPath);
    const log = (await fs.readFile(logPath, {encoding: 'utf8', flag: 'r'}))
        .split('\n')
        .filter(v => v)
        .map(v => v.split(' '));
    let i = log.findIndex((v, i) => v[0] != remote[i]);

    if (i != remote.length) {
        console.warn('huh. log has changed in non-ff way');
        await noms('ds', '-d', `${dbPath}::${REMOTE_BRANCH}`);
        i = 0;
    }

    // For each remaining commit in the log, we may already have it locally (eg if we ourselves pushed it).
    // Otherwise, we have to build it by replaying.

    const local = await getLog(dbPath, LOCAL_BRANCH);
    for (let l; l = log[i]; i++) {
        const [commitRef, opName, ...opArgs] = l;
        if (!local.indexOf(commitRef) > -1) {
            await runOp(dbPath, REMOTE_BRANCH, opName, JSON.parse(opArgs.join(' ')));
        }
    }
}

async function rebase(dbPath) {
    const local = await getLog(dbPath, LOCAL_BRANCH);
    const remote = await getLog(dbPath, REMOTE_BRANCH);

    // Find place where remote and local branch diverge
    let i = local.findIndex((v, idx) => v != remote[idx]);
    
    // If this spot is the end of remote branch, then nothing to do, this is a fast forward.
    if (i == remote.length) {
        console.log("fast-forward - nothing to do");
        return;
    }

    // otherwise:
    // - replay each operation onto a temporary branch
    // - update local when done
    await deleteBranch(dbPath, 'tmp');
    await noms('sync', `${dbPath}::${REMOTE_BRANCH}`, `${dbPath}::tmp`);
    let ref;
    for (let l; l = local[i]; i++) {
        const [name, args] = await getOpFromCommit(dbPath, l);
        await runOp(dbPath, "tmp", name, args);
    }
    await noms('sync', `${dbPath}::tmp`, `${dbPath}::${LOCAL_BRANCH}`);
    await deleteBranch(dbPath, 'tmp');
}

async function runOp(dbName, branch, opName, args) {
    console.log('Running', opName, args, 'against', dbName, branch)
    const db = new Database(dbName, branch);
    const op = ops.find(o => o.name == opName);
    if (!op) {
        throw new Error('Unknown op: ' + opName);
    }
    await op(db, ...args);
    return await commit(db, branch, opName, args);
}

async function commit(db, branch, opName, args) {
    const val = db.root_ || await db.get();
    const f = await tmp.file();
    await fs.writeFile(f.path, JSON.stringify(val));
    const f2 = await tmp.file();
    await fs.writeFile(f2.path, JSON.stringify(args));
    const jsonRef = await noms('json', 'in', db.path_, f.path);
    const argsRef = await noms('json', 'in', db.path_, f2.path);
    const metaRef = await noms('struct', 'new', db.path_, 'name', opName, 'args', `@${argsRef}`);
    await noms('commit', '--allow-dupe=1', '--meta-p', `op=${metaRef}`, `'${jsonRef}'`, `${db.path_}::${branch}`);
    const [noDate] = (await noms('struct', 'del', `${db.path_}::${branch}.meta`, 'date')).split('.');
    await noms('sync', `${db.path_}::${noDate}`, `${db.path_}::${branch}`);
    return noDate;
}

async function noms(...args) {
    const cmd = ['noms'].concat(args).join(' ');
    console.log(cmd);
    const { stdout: r } = await pexec(cmd);
    return r.trim();
}

async function getLog(dbPath, branch) {
    if (!await hasBranch(dbPath, branch)) {
        return [];
    }
    return (await noms('log', '--oneline', `${dbPath}::${branch}`)).split('\n')
        .map(line => line.split(' ')[0])
        .reverse();
}

async function deleteBranch(dbPath, branch) {
    if (await hasBranch(dbPath, branch)) {
        await noms('ds', '-d', `${dbPath}::${branch}`);
    }
}

async function hasBranch(dbPath, branch) {
    const datasets = await noms('ds', dbPath);
    return datasets.indexOf(branch) > -1;
}

async function getOpFromCommit(dbPath, ref) {
    return (await Promise.all([
        noms('show', `${dbPath}::#${ref}.meta.op.name`),
        noms('json', 'out', '--indent=""', `${dbPath}::#${ref}.meta.op.args`, '@'),
    ])).map(s => JSON.parse(s));
}

module.exports = {Database, opCmd, push, pull, rebase};
