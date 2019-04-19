const fs = require('fs').promises;
const tmp = require('tmp-promise');
const util = require('util');
const touch = require('touch');
const { exec, spawn } = require('child_process');
const pexec = util.promisify(exec);

const LOCAL_BRANCH = 'local';

class Database {
    constructor(path) {
        this.path_ = path;
        this.root_ = null;
    }

    async get() {
        const datasets = await noms('ds', this.path_);
        if (datasets.indexOf(LOCAL_BRANCH) == -1) {
            return {};
        }
        return JSON.parse(await noms('json', 'out', `${this.path_}::${LOCAL_BRANCH}.value`, '@'));
    }

    set(root) {
        this.root_ = root;
    }
}

async function commit(db, opName, args) {
    const val = db.root_ || await db.get();
    const f = await tmp.file();
    await fs.writeFile(f.path, JSON.stringify(val));
    const jsonRef = await noms('json', 'in', db.path_, f.path);
    const metaRef = await noms('struct', 'new', db.path_, 'name', opName, 'args', JSON.stringify(args));
    await noms('commit', '--meta-p', `op=${metaRef}`, `'${jsonRef}'`, `${db.path_}::${LOCAL_BRANCH}`);
    const [noDate] = (await noms('struct', 'del', `${db.path_}::${LOCAL_BRANCH}.meta`, 'date')).split('.');
    await noms('sync', `${db.path_}::${noDate}`, `${db.path_}::${LOCAL_BRANCH}`);
}

async function noms(...args) {
    const cmd = ['noms'].concat(args).join(' ');
    console.log(cmd);
    const { stdout: r } = await pexec(cmd);
    return r.trim();
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

module.exports = {Database, commit, push};
