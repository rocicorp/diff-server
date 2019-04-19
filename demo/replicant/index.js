#!/usr/bin/env node

const program = require('commander');
const {Database, commit} = require('./db');

program
    .command('op <db> <name> [args...]')
    .description('Run an op against the current client state')
    .action(opCmd);

async function opCmd(dbName, opName, args) {
    const db = new Database(dbName);
    const op = ops[opName];
    if (!op) {
        throw new Error('Unknown op: ' + opName);
    }
    await op(db, ...args);
    await commit(db, opName, args);
}

const ops = {
    'setColor': setColor,
};

async function setColor(db, name) {
    const val = await db.get();
    val.color = name;
    db.set(val);
}

program.parse(process.argv);
