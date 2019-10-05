import { NativeModules } from 'react-native';

const { Replicant: repm } = NativeModules;

export default class Replicant {
  onChange = () => {};
  onSync = (syncing) => {};
  _timerID = -1;

  constructor(remote) {
      this._remote = remote;
      this._root = this._invoke('open', {name: remote})
        .then(() => this._getRoot())
        // open can fail if the database is already open, which can
        // happen especially during development when hot-reloading.
        // just ignore this.
        .catch(() => this._getRoot());

        this._root
        .then(() => this.sync());
  }

  async root() {
      return await this._root;
  }

  // Adds new transactions to the db.
  async putBundle(bundle) {
    // We check for changes here, even though putBundle doesn't change data, because
    // it can change the bundle which the client app uses to read the data, thus it
    // can affect display.
    return this._result(await this._checkChange(await this._invoke('putBundle', {'code': bundle})));
  }

  // Executes the named function with provided arguments from the current
  // bundle as an atomic transaction.
  async exec(functionName, args) {
    if (args == null) {
      args = [];
    }
    return this._result(await this._checkChange(await this._invoke('exec', {'name': functionName, 'args': args})));
  }

  // Puts a single value into the database in its own transaction.
  async put(id, value) {
    return this._result(await this._checkChange(await this._invoke('put', {'id': id, 'value': value})));
  }

  // Get a single value from the database.
  async get(id) {
    return this._result(await this._invoke('get', {'id': id}));
  }

  // Gets many values from the database.
  async scan(prefix, startAtID, limit) {
    return await this._invoke('scan', {prefix, startAtID, limit});
  }

  // Synchronizes the database with the server. New local transactions that have been executed since the last
  // sync are sent to the server, and new remote transactions are received and replayed.
  async sync() {
    this._fireOnSync(true);
    try {
      if (this._timerID == 0) {
        // Another call stack is already inside _sync();
        return;
      }

      clearTimeout(this._timerID);
      this._timerID = 0;
      await this._checkChange(await this._invoke("sync", {'remote': this._remote}));
    } catch (e) {
      console.warn('ERROR DURING SYNC', e.toString());
    } finally {
      this._timerID = setTimeout(() => this.sync(), 5000);
      this._fireOnSync(false);
    }
  }

  async dropDatabase() {
    return this._result(await this._checkChange(await this._invoke('dropDatabase', {})));
  }

  async _getRoot() {
    var res = await this._invoke('getRoot', {});
    return res['root'];
  }

  _result(m) {
    return m == null ? null : m['result'];
  }

  async _checkChange(result) {
    var currentRoot = await this._root;  // instantaneous except maybe first time
    if (result != null && result['root'] != null && result['root'] != currentRoot) {
      this._root = Promise.resolve(result['root']);
      this._fireOnChange();
    }
    return result;
  }

  async _invoke(name, args) {
    const r = await repm.dispatch(this._remote, name, JSON.stringify(args));
    return r == '' ? null : JSON.parse(r);
  }

  _fireOnChange() {
    if (this.onChange != null) {
      try {
        this.onChange();
      } catch (e) {
        console.error(e);
      }
    }
  }

  _fireOnSync(syncing) {
    if (this.onSync != null) {
      try {
        this.onSync(syncing);
      } catch (e) {
        console.error(e);
      }
    }
  }
};
