function codeVersion() {
    return 3;
}

function init() {
    var schemaVersion = db.get('schemaVersion') || 0;
    if (schemaVersion < 1) {
        db.put('schemaVersion', 1);
        db.put('todos', {});
    }
}

function addTodo(id, title, order) {
    var todos = _read();
    todos[id] = {
        title: title,
        done: false,
        order: order,
    };
    _store(todos);
}

function setDone(id, done) {
    var todos = _read();
    var item = todos[id];
    if (!item) {
        throw new Error("todo not found: " + id);
    }
    item.done = done;
    _store(todos);
}

function getTodos() {
    return _read();
}

function deleteTodo(id) {
    var todos = _read();
    delete todos[id];
    _store(todos);
}

function deleteAllTodos() {
    db.put('todos', {});
}

function _read() {
    return db.get('todos');
}

function _store(todos) {
    db.put('todos', todos);
}
