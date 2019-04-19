module.exports = [
    setColor,
    toggleColor,
    append,
    goDogGo,
    stockWidgets,
    sellWidget,
];

async function setColor(db, name) {
    const val = await db.get();
    val.color = name;
    db.set(val);
}

async function toggleColor(db) {
    const val = await db.get();
    val.color = val.color == 'red' ? 'green' : 'red';
    db.set(val);
}

async function append(db, text) {
    const val = await db.get();
    val.words = val.words || [];
    val.words.push(text);
    db.set(val);
}

async function goDogGo(db) {
    const val = await db.get();
    await append(db, `Go dog go, the light is ${val.color || 'red'} now!`);
}

async function stockWidgets(db, inc) {
    const val = await db.get();
    val.widgets = (val.widgets || 0) + parseInt(inc);
    db.set(val);
}

async function sellWidget(db) {
    const val = await db.get();
    if (val.widgets) {
        val.widgets--;
    }
    db.set(val);
}
