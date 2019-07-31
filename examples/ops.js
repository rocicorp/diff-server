function setColor(name) {
    db.put('color', name);
}

function toggleColor() {
    var val = db.get('color');
    val = val == 'red' ? 'green' : 'red';
    db.put('color', val);
}

function append(key, text) {
    var val = db.get(key) || [];
    val.push(text);
    db.put(key, val);
}

function stockWidgets(inc) {
    var val = db.get('widgets');
    if (val === undefined) {
        val = 0;
    }
    val += parseInt(inc);
    db.put('widgets', val);
}

function sellWidget() {
    var val = db.get('widgets');
    if (val) {
        val--;
        db.put('widgets', val);
    }
}

function index(text) {
    var sorted = db.get('sorted');
    if (!sorted) {
        sorted = [];
    }
    // imagine binary search + splice :).
    sorted.push(text);
    sorted.sort();
    db.put('sorted', sorted);
}
