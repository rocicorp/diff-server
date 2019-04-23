const fs = require('fs').promises;
const tmp = require('tmp-promise');
const util = require('util');
const touch = require('touch');
const { exec, spawn } = require('child_process');
const pexec = util.promisify(exec);
const _ = require('underscore');
const program = require('commander');

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
    await runOp(dbName, LOCAL_BRANCH, dbName, opName, args);
}

async function push(dbPath, logPath) {
    const local = await branchHistory(dbPath, LOCAL_BRANCH);
    await touch(logPath);
    const log = await serverLog(logPath);
    const forkPoint = _.zip(local, log.map(([ref]) => ref))
        .findIndex(([localRef, logRef]) => localRef != logRef);
    if (forkPoint == -1) {
        console.log('Server is already up to date. Nothing to do.')
        return;
    }
    if (forkPoint != log.length) {
        console.error("Non fast-forward push not allowed. Pull first.");
        return;
    }

    const f = await fs.open(logPath, 'a');
    for (let i = forkPoint, l; l = local[i]; i++) {
        const [source, name, args] = await getOpFromCommit(dbPath, l);
        await f.writeFile([l, source, name, JSON.stringify(args)].join(' ') + '\n');
    }
    await f.close();
}

async function pull(dbPath, logPath) {
    // find place where remote branch and log diverge
    const remote = await branchHistory(dbPath, REMOTE_BRANCH);
    await touch(logPath);
    const log = await serverLog(logPath);

    const forkPoint = _.zip(remote, log.map(([ref]) => ref))
        .findIndex(([remoteRef, logRef]) => remoteRef != logRef);
    if (forkPoint == -1) {
        console.log("remote is unchanged - nothing to do");
        return;
    }
    if (forkPoint != remote.length) {
        console.error('eep: remote has changed in non-ff way');
        return;
    }

    // For each remaining commit in the log, we may already have it locally (eg if we ourselves pushed it).
    // Otherwise, we have to build it by replaying.
    for (let i = forkPoint, l; l = log[i]; i++) {
        const [commitRef, source, opName, ...opArgs] = l;
        if (await haveRef(dbPath, commitRef)) {
            await noms('sync', `${dbPath}::#${commitRef}`, `${dbPath}::${REMOTE_BRANCH}`)
        } else {
            await runOp(dbPath, REMOTE_BRANCH, source, opName, JSON.parse(opArgs.join(' ')));
        }
    }
}

async function rebase(dbPath) {
    const local = await branchHistory(dbPath, LOCAL_BRANCH);
    const remote = await branchHistory(dbPath, REMOTE_BRANCH);

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
        const [source, name, args] = await getOpFromCommit(dbPath, l);
        await runOp(dbPath, "tmp", source, name, args);
    }
    await noms('sync', `${dbPath}::tmp`, `${dbPath}::${LOCAL_BRANCH}`);
    await deleteBranch(dbPath, 'tmp');
}

async function runOp(dbName, branch, source, opName, args) {
    console.log('Running', opName, args, 'against', dbName, branch)
    const db = new Database(dbName, branch);
    const op = ops.find(o => o.name == opName);
    if (!op) {
        throw new Error('Unknown op: ' + opName);
    }
    await op(db, ...args);
    return await commit(db, branch, source, opName, args);
}

async function commit(db, branch, source, opName, args) {
    const val = db.root_ || await db.get();
    const f = await tmp.file();
    await fs.writeFile(f.path, JSON.stringify(val));
    const f2 = await tmp.file();
    await fs.writeFile(f2.path, JSON.stringify(args));
    const jsonRef = await noms('json', 'in', db.path_, f.path);
    const argsRef = await noms('json', 'in', db.path_, f2.path);
    const metaRef = await noms('struct', 'new', db.path_, 'name', opName, 'args', `@${argsRef}`, 'source', source);
    await noms('commit', '--allow-dupe=1', '--meta-p', `op=${metaRef}`, `'${jsonRef}'`, `${db.path_}::${branch}`);
    const [noDate] = (await noms('struct', 'del', `${db.path_}::${branch}.meta`, 'date')).split('.');
    await noms('sync', `${db.path_}::${noDate}`, `${db.path_}::${branch}`);
    return noDate;
}

async function noms(...args) {
    const cmd = ['noms'].concat(args).join(' ');
    if (program.verbose) {
        console.log(cmd);
    }
    const { stdout: r } = await pexec(cmd);
    return r.trim();
}

async function branchHistory(dbPath, branch) {
    if (!await hasBranch(dbPath, branch)) {
        return [];
    }
    return (await noms('log', '--oneline', `${dbPath}::${branch}`)).split('\n')
        .map(line => line.split(' ')[0])
        .reverse();
}

async function serverLog(logPath) {
    return (await fs.readFile(logPath, {encoding: 'utf8', flag: 'r'}))
        .split('\n')
        .filter(v => v)
        .map(v => v.split(' '));
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
        noms('show', `${dbPath}::#${ref}.meta.op.source`),
        noms('show', `${dbPath}::#${ref}.meta.op.name`),
        noms('json', 'out', '--indent=""', `${dbPath}::#${ref}.meta.op.args`, '@'),
    ])).map(s => JSON.parse(s));
}

async function haveRef(dbPath, ref) {
    return await noms('show', `${dbPath}::#${ref}`);
}

module.exports = {Database, opCmd, push, pull, rebase};
