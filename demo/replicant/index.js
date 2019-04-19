#!/usr/bin/env node

const program = require('commander');
const {opCmd, push, pull} = require('./db');

program
    .command('op <db> <name> [args...]')
    .description('Run an op against the current client state')
    .action(opCmd);

program
    .command('push <db> <log>')
    .description('Pushes new local ops to the server')
    .action(push);

program
    .command('pull <db> <log>')
    .description('Pulls remote ops from server')
    .action(pull);

program.parse(process.argv);
