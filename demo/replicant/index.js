#!/usr/bin/env node

const program = require('commander');
const {opCmd, push, pull, rebase, sync} = require('./db');
const {opReg, list} = require('./reg.js');

program
    .option('-v,--verbose', 'Print verbosely to the console', false)
    .option('-ops,--opsfile', 'File to store op definitions in', 'ops.json')

program
    .command('reg')
    .description('Register a new op from the code in stdin')
    .action(opReg);

program
    .command('list')
    .description('Lists all registered ops')
    .action(list);

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

program
    .command('rebase <db>')
    .description('Rebase local changes onto remote')
    .action(rebase);

program
    .command('sync <db> <log>')
    .description('pull && rebase && push')
    .action(sync)

program.parse(process.argv);
if (!program.args.length) {
    program.help();
}
