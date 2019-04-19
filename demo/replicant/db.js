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
        const datasets = await noms('ds', this.path_);
        if (datasets.indexOf(this.branch_) == -1) {
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
    const local = (await noms('log', '--oneline', `${dbPath}::${LOCAL_BRANCH}`)).split('\n')
        .map(line => line.split(' ')[0])
        .reverse();
    await touch(logPath);
    const remote = (await fs.readFile(logPath, {encoding: 'utf8', flag: 'r'})).split('\n');
    let i = remote.findIndex((v, i) => v.split(' ')[0] != local[i]);
    const f = await fs.open(logPath, 'a');
    for (let l; l = local[i]; i++) {
        const [name, args] = (await Promise.all([
            noms('show', `${dbPath}::#${l}.meta.op.name`),
            noms('show', `${dbPath}::#${l}.meta.op.args`),
        ])).map(s => s.substr(1, s.length - 2));
        await f.writeFile([l, name, args].join(' ') + '\n');
    }
    await f.close();
}

async function pull(dbPath, logPath) {
    // find place where remote branch and log diverge
    let remote = [];
    try {
        remote = (await noms('log', '--oneline', `${dbPath}::${REMOTE_BRANCH}`)).split('\n')
            .map(line => line.split(' ')[0])
            .reverse();
    } catch (e) {
    }

    await touch(logPath);
    const log = (await fs.readFile(logPath, {encoding: 'utf8', flag: 'r'}))
        .split('\n')
        .filter(v => v)
        .map(v => v.split(' '))
        .reverse();
    let i = log.findIndex((v, i) => v[0] != remote[i]);

    if (i != remote.length) {
        console.warn('huh. log has changed in non-ff way');
        await noms('ds', '-d', `${dbPath}::${REMOTE_BRANCH}`);
        i = 0;
    }

    // For each remaining commit in the log, we may already have it locally (eg if we ourselves pushed it).
    // Otherwise, we have to build it by replaying.

    const local = (await noms('log', '--oneline', `${dbPath}::${LOCAL_BRANCH}`)).split('\n')
        .map(line => line.split(' ')[0])
        .reverse();

    for (let l; l = log[i]; i++) {
        const [commitRef, opName, ...opArgs] = l;
        if (!local.indexOf(commitRef) > -1) {
            await runOp(dbPath, REMOTE_BRANCH, opName, JSON.parse(opArgs.join(' ')));
        }
    }
}

async function rebase(dbPath) {
    // if head of remote exists in local, then nothing to do (local is a ff)
    // otherwise:
    // - replay each operation onto a temporary branch
    // - update local when done
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
    const jsonRef = await noms('json', 'in', db.path_, f.path);
    const metaRef = await noms('struct', 'new', db.path_, 'name', opName, 'args', JSON.stringify(args));
    await noms('commit', '--meta-p', `op=${metaRef}`, `'${jsonRef}'`, `${db.path_}::${branch}`);
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

module.exports = {Database, opCmd, push, pull};
