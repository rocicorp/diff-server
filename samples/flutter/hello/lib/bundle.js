function codeVersion() {
    return 1.1;
}

function incr(delta) {
    var val = getCount();
    db.put('count', val + delta);
}

function getCount() {
    return db.get('count') || 0;
}
