#!/usr/bin/env node

const program = require('commander');
const {Database, commit, push} = require('./db');
const ops = require('./ops');

program
    .command('op <db> <name> [args...]')
    .description('Run an op against the current client state')
    .action(opCmd);

program
    .command('push <db> <log>')
    .description('Pushes new local ops to the server')
    .action(push);

async function opCmd(dbName, opName, args) {
    const db = new Database(dbName);
    const op = ops.find(o => o.name == opName);
    if (!op) {
        throw new Error('Unknown op: ' + opName);
    }
    await op(db, ...args);
    await commit(db, opName, args);
}

program.parse(process.argv);
