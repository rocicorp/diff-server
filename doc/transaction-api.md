# JavaScript Transaction API

Replicant transactions run in a clean ES5 environment (ES6 coming soon). This means that the entire
JavaScript language is available, but no browser or npm objects like `window`, `document`, `process`, etc.

## Transaction Function Restrictions

Transaction functions must be [pure](https://en.wikipedia.org/wiki/Pure_function): they should always make
the same changes to the database given the same parameters.

Don't use things like `Math.random()` or `Date.now()` inside transactions because these generate a different
value everytime they call. Instead, pass these values into functions as parameters. See also [issue 15](#15).

## The Database Object

Transaction functions have access to a special `db` variable in the global scope with the following API:

#### put(id, value)

* `id` *string*
* `value` Any JSON-compatible value, except `null`.

Puts an entry into the database.

#### has(id)

* `returns` *bool*
* `id` *string*

Check whether an entry exists in the database.

#### get(id)

* `returns` Any JSON-compatible value, except `null`.

Gets an entry out of the database.

#### scan(options)

* `returns` *List<Entry>` where `Entry` is:
  * `id` *string* - The ID of the entry
  * `value` Any JSON-compatible value
* `options`
  * `prefix` *optional string* - Filter returned values to those whose ID have this prefix
  * `startAtID` *optional string* - Return values starting with the specified ID (inclusive)
  * `startAfterID` *optional string* - Return values starting after the specified ID (exclusive)
  * `limit` *optional int* - Return at most this many items

Returns many entries from the database, sorted by their ID.

#### del(id)

* `id` *string*

Deletes an entry from the database. If the entry is not present, does nothing.
