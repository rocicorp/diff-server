const fs = require('fs').promises;
const tmp = require('tmp-promise');
const util = require('util');
const { exec, spawn } = require('child_process');
const pexec = util.promisify(exec);

const LOCAL_BRANCH = 'local';

class Database {
    constructor(path) {
        this.path_ = path;
        this.root_ = null;
    }

    async get() {
        const datasets = await run('ds', this.path_);
        if (datasets.indexOf(LOCAL_BRANCH) == -1) {
            return {};
        }
        return JSON.parse(await run('json', 'out', `${this.path_}::${LOCAL_BRANCH}.value`, '@'));
    }

    set(root) {
        this.root_ = root;
    }
}

async function commit(db, opName, args) {
    const val = db.root_ || await db.get();
    const f = await tmp.file();
    await fs.writeFile(f.path, JSON.stringify(val));
    const jsonRef = await run('json', 'in', db.path_, f.path);
    const metaRef = await run('struct', 'new', db.path_, 'name', opName, 'args', JSON.stringify(args));
    await run('commit', '--meta-p', `op=${metaRef}`, `'${jsonRef}'`, `${db.path_}::${LOCAL_BRANCH}`);
    /*
    db.root_ = null;
    */
}

async function run(...args) {
    const cmd = ['noms'].concat(args).join(' ');
    console.log(cmd);
    const { stdout: r } = await pexec(cmd);
    return r.trim();
}

module.exports = {Database, commit};

