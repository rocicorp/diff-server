import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:uuid/uuid.dart';

import 'model.dart';
import 'replicant.dart';
import 'settings.dart';

const bundleVersion = 2;

void main() => runApp(MyApp());

class MyApp extends StatelessWidget {
  // This widget is the root of your application.
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Todo',
      theme: ThemeData(
        primarySwatch: Colors.blue,
      ),
      home: MyHomePage(),
    );
  }
}

class MyHomePage extends StatefulWidget {
  @override
  _MyHomePageState createState() => _MyHomePageState();
}

class _MyHomePageState extends State<MyHomePage> {
  static final _replicant = Replicant();

  final GlobalKey<ScaffoldState> _scaffoldKey = new GlobalKey<ScaffoldState>();

  List<Todo> _todos = [];
  Timer _timer;

  _MyHomePageState() {
    _init();
  }

  @override
  Widget build(BuildContext context) {
    return new Scaffold(
      key: _scaffoldKey,
      appBar: new AppBar(
         title: new Text('Todo List')
      ),
      drawer: TodoDrawer(_sync, _dropDatabase),
      body: TodoList(_todos, _handleDoneChanged),
      floatingActionButton: new FloatingActionButton(
        onPressed: _pushAddTodoScreen,
        tooltip: 'Add task',
        child: new Icon(Icons.add)
      ),
    );
  }

  Future<void> _init() async {
    await _registerBundle();
    await _replicant.exec('init');
    await _load();
    await _sync(force: true);
  }

  Future<void> _load() async {
    final Map<String, dynamic> res = await _replicant.exec('getTodos');
    List<Todo> todos = List.from(res.entries.map((e) => Todo.fromJson(e.key, e.value)));
    todos.sort((t1, t2) => t1.order < t2.order ? -1 : t1.order == t2.order ? 0 : 1);
    setState(() {
      _todos = todos;
    });
  }

  Future<void> _handleDoneChanged(String id, bool isDone) async {
    await _replicant.exec('setDone', [id, isDone]);
    _load();
  }

  Future<void> _registerBundle() async {
    var registeredVersion = 0;
    try {
      registeredVersion = await _replicant.exec('codeVersion');
    } catch (e) {
      // https://github.com/aboodman/replicant/issues/25
      if (!e.toString().contains("Unknown function: codeVersion")) {
        throw e;
      }
    }

    if (registeredVersion < bundleVersion) {
      await _replicant.putBundle(await rootBundle.loadString('assets/bundle.js', cache: false));
      print("Upgraded bundle version from $registeredVersion to $bundleVersion");
    }
  }

  Future<void> _sync({force:false, status:false}) async {    
    if (_timer == null) {
      if (!force) {
        // Another call stack is already inside _sync();
        return;
      }
    } else {
      _timer.cancel();
    }
    
    try {
      _timer = null;
      await _replicant.sync(db);
      await _load();
    } catch (e) {
      print('ERROR DURING SYNC');
      print(e);
      // We are seeing some consistency errors during sync -- we push commits,
      // then turn around and fetch them and expect to see them, but don't.
      // that is bad, but for now, just retry.
      _timer = new Timer(new Duration(milliseconds: 100), _sync);
    } finally {
      _timer = new Timer(new Duration(seconds: 5), _sync);
    }
  }

  Future <void> _dropDatabase() async {
    Navigator.pop(context);
    await _replicant.exec('deleteAllTodos');
    await _init();
  }

  void _addTodoItem(String task) async {
    var uuid = new Uuid();
    // Only add the task if the user actually entered something
    if(task.length > 0) {
      await _replicant.exec('addTodo', [uuid.v4(), task, _todos.length]);
      await _load();
      await _sync(status: true);
    }
  }

  void _removeTodoItem(int index) {
    //setState(() => _todoItems.removeAt(index));
  }

  void _pushAddTodoScreen() {
    // Push this page onto the stack
    Navigator.of(context).push(
      // MaterialPageRoute will automatically animate the screen entry, as well as adding
      // a back button to close it
      new MaterialPageRoute(
        builder: (context) {
          return new Scaffold(
            appBar: new AppBar(
              title: new Text('Add a new task')
            ),
            body: new TextField(
              autofocus: true,
              onSubmitted: (val) {
                _addTodoItem(val);
                Navigator.pop(context); // Close the add todo screen
              },
              decoration: new InputDecoration(
                hintText: 'Enter something to do...',
                contentPadding: const EdgeInsets.all(16.0)
              ),
            )
          );
        }
      )
    );
  }
}

class TodoList extends StatelessWidget {
  final List<Todo> _todos;
  final Future<void> Function(String, bool) _handleDoneChange;

  TodoList(this._todos, this._handleDoneChange);

  // Build the whole list of todo items
  @override
  Widget build(BuildContext build) {
    return ListView.builder(
      itemBuilder: (BuildContext _context, int index) {
        // itemBuilder will be automatically be called as many times as it takes for the
        // list to fill up its available space, which is most likely more than the
        // number of todo items we have. So, we need to check the index is OK.
        if(index < _todos.length) {
            var todo = _todos[index];
            return new CheckboxListTile (
              title: new Text(todo.title),
              value: todo.done,
              onChanged: (bool newValue) {
                _handleDoneChange(todo.id, newValue);
              },
            );
        }
      },
    );
  }

  void _handleRemove(int index) {
    /*
    showDialog(
      context: context,
      builder: (BuildContext context) {
        return new AlertDialog(
          title: new Text('Mark "${_todoItems[index]}" as done?'),
          actions: <Widget>[
            new FlatButton(
              child: new Text('CANCEL'),
              // The alert is actually part of the navigation stack, so to close it, we
              // need to pop it.
              onPressed: () => Navigator.of(context).pop()
            ),
            new FlatButton(
              child: new Text('MARK AS DONE'),
              onPressed: () {
                _removeTodoItem(index);
                Navigator.of(context).pop();
              }
            )
          ]
        );
      }
    );
    */
  }
}

class TodoDrawer extends StatelessWidget {
  final Future<void> Function({bool force, bool status}) _sync;
  final Future<void> Function() _dropDatabase;

  TodoDrawer(this._sync, this._dropDatabase);

  @override
  Widget build(BuildContext context) {
    return Drawer(
      child: ListView(
        children: <Widget>[
          DrawerHeader(
            child: Text(""),
              decoration: BoxDecoration(
              color: Colors.blue,
            ),
          ),
          ListTile(
            title: Text('Sync'),
            onTap: () => _sync(status: true),
          ),
          ListTile(
            title: Text('Delete local state'),
            onTap: _dropDatabase,
          ),
        ],
      ),
    );
  }
}
