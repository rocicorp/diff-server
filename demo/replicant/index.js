#!/usr/bin/env node

const program = require('commander');
const {opCmd, push, pull, rebase} = require('./db');

program
    .option('-v,--verbose', 'Print verbosely to the console', false);

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


program.parse(process.argv);
if (!program.args.length) {
    program.help();
}
