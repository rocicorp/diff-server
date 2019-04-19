#!/usr/bin/env node

const program = require('commander');
const {Database, commit} = require('./db');

program
    .command('op <db> <name> [args...]')
    .description('Run an op against the current client state')
    .action(opCmd);

async function opCmd(dbName, opName, args) {
    const db = new Database(dbName);
    const op = ops.find(o => o.name == opName);
    if (!op) {
        throw new Error('Unknown op: ' + opName);
    }
    await op(db, ...args);
    await commit(db, opName, args);
}

const ops = [
    setColor,
    toggleColor,
    sum,
    append,
    goDogGo,
];

async function setColor(db, name) {
    const val = await db.get();
    val.color = name;
    db.set(val);
}

async function toggleColor(db) {
    const val = await db.get();
    val.color = val.color == 'red' ? 'green' : 'red';
    db.set(val);
}

async function sum(db, inc) {
    const val = await db.get();
    val.total = val.total || 0;
    val.total += parseInt(inc);
    db.set(val);
}

async function append(db, text) {
    const val = await db.get();
    val.words = val.words || [];
    val.words.push(text);
    db.set(val);
}

async function goDogGo(db) {
    const val = await db.get();
    await append(db, `Go dog go, the light is ${val.color || 'red'} now!`);
}

program.parse(process.argv);
