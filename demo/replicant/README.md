# Replicant Demo

This is a super sketchy fast ugly duct-tape, glue, and chewing gum prototype of Replicant. This is not beautiful code.

Specifically, what we have here is a CLI program that you should think of as a prototype of the eventual Replicant API. So in real life, Replicant is not going to take the form of a CLI (or at least
not *only* a CLI). It will have in-process bindings for various languages. But it was expedient to prototype it that way
here.

## Setup

1. [Install Noms](https://github.com/attic-labs/noms#install). You will need a pretty recent version.
2. Install this app using `yarn install`.

## Usage

```bash
# Register transactions (available functions that can be run from client code) in the database.
# Assuming you're on MacOS, you can copy some sample functions from ops.js onto your clipboard, then:
pbpaste | replicant reg

# List the registered transactions
replicant list

# Run some transactions on a local file-backed database in this directory called "db1"
# (these op names correspond to the transactions in ops.js - you will need to register them first)
replicant op db1 setColor green
replicant op db1 insert aaron
replicant op db1 insert susan
replicant op db1 stockWidgets 1
replicant op db1 sellWidget

# See state of db1 after these changes
noms show db1::local.value

# Run some (potentially conflicting) transactions on db2
# Writes to a single register. LWW.
replicant op db2 setColor red
# Sets a value in the db dependent upon another value. After merge, the two edits still need to match.
replicant op db2 dog
# Inserts into a sorted list. After merge, the list should be merged and still sorted.
replicant op db2 insert abby
replicant op db2 insert sam
# Sells a non-existent widget. After merge, the total number of widgets must be zero. We can't sell the same widget twice.
replicant op db2 sellWidget

# See state of db2 after these changes, before merge
noms show db2::local.value

# Sync db1 with the server
replicant sync db1 server.txt

# Sync db2 with the server
replicant sync db2 server.txt

# Sync the changes from db2 back to db1
replicant sync db1 server.txt

# The two databases now match and the merges have all been handled naturally
noms show db1::local.value
noms show db2::local.value

# In particular:
# - Both databases have the color green, and the dependent write also reflects "green".
# - The list of sorted names is merged and sorted correctly
# - The number of widgets is zero, not -1
```
