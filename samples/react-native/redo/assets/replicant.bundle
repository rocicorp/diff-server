function codeVersion() {
    return 6.2;
}

function init() {
    var schemaVersion = db.get('schemaVersion') || 0;
    if (schemaVersion < 4) {
        db.del('todos');
        db.put('schemaVersion', schemaVersion);
    }
}

function addTodo(id, title, order) {
    _write(id, {
        title: title,
        done: false,
        order: order,
    });
}

function getAllTodos() {
    return db.scan({prefix: prefix}).map(function(item) {
        item.id = _tid(item.id);
        return item;
    });
}

function setDone(id, done) {
    var item = _read(id);
    if (!item) {
        return;
    }
    item.done = done;
    _write(id, item);
}

function setOrder(id, order) {
    var item = _read(id);
    if (!item) {
        return;
    }
    item.order = order;
    _write(id, item);
}

function deleteTodo(id) {
    db.del(_fullid(id));
}

function deleteAllTodos() {
    db.scan({prefix: prefix}).forEach(function(item) {
        db.del(item.id);
    });
}

function _read(id) {
    return db.get(_fullid(id));
}

function _write(id, todo) {
    db.put(_fullid(id), todo);
}

var prefix = 'todo/';

function _fullid(tid) {
    return prefix + tid;
}

function _tid(fullid) {
    return fullid.substr(prefix.length);
}
