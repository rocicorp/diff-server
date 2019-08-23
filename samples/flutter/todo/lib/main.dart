import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:uuid/uuid.dart';

import 'replicant.dart';

const bundleVersion = 1;

void main() => runApp(MyApp());

class MyApp extends StatelessWidget {
  // This widget is the root of your application.
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Flutter Demo',
      theme: ThemeData(
        primarySwatch: Colors.blue,
      ),
      home: MyHomePage(title: 'ToDo List'),
      
    );
  }
}

class MyHomePage extends StatefulWidget {
  MyHomePage({Key key, this.title}) : super(key: key);

  // This widget is the home page of your application. It is stateful, meaning
  // that it has a State object (defined below) that contains fields that affect
  // how it looks.

  // This class is the configuration for the state. It holds the values (in this
  // case the title) provided by the parent (in this case the App widget) and
  // used by the build method of the State. Fields in a Widget subclass are
  // always marked "final".

  final String title;

  @override
  _MyHomePageState createState() => _MyHomePageState();
}

class _MyHomePageState extends State<MyHomePage> {
  static final _replicant = Replicant('replicant.dev/samples/todo');

  _MyHomePageState() {
    _init();
  }

  List<dynamic> _todoItems = [];

  Future<void> _init() async {
    await _registerBundle();
    await _replicant.exec('init', []);
    await _getTodos();
  }

  Future<void> _registerBundle() async {
    /*
    var registeredVersion = 0;
    try {
      registeredVersion = await _replicant.exec('codeVersion');
    } catch (e) {
      print(e.toString());
      // https://github.com/aboodman/replicant/issues/25
      if (!e.toString().contains("Unknown function: codeVersion")) {
        throw e;
      }
    }

    if (registeredVersion < bundleVersion) {
      */
      await _replicant.putBundle(await rootBundle.loadString('assets/bundle.js', cache: false));
      //print("Upgraded bundle version from $registeredVersion to $bundleVersion");
    //}
  }

  Future<void> _sync() async {
    print("Syncing...");
    await _replicant.sync('https://replicate.to/serve/boodman-todos');
    print("Done");
  }

  Future <void> _dropDatabase() async {
    await _replicant.dropDatabase();
    await _init();
    Navigator.pop(context);
  }

  Widget _buildDrawer() {
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
            onTap: _sync,
          ),
          ListTile(
            title: Text('Delete local state'),
            onTap: _dropDatabase,
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return new Scaffold(
      appBar: new AppBar(
         title: new Text('Todo List')
      ),
      drawer: _buildDrawer(),
      body: _buildTodoList(),
      floatingActionButton: new FloatingActionButton(
        onPressed: _pushAddTodoScreen,
        tooltip: 'Add task',
        child: new Icon(Icons.add)
      ),
    );
  }

  void _addTodoItem(String task) async {
    var uuid = new Uuid();
    // Only add the task if the user actually entered something
    if(task.length > 0) {
      await _replicant.exec('addTodo', [uuid.v4(), task, 5]);
      await _getTodos();
    }
  }

  void _removeTodoItem(int index) {
    //setState(() => _todoItems.removeAt(index));
  }

  void _promptRemoveTodoItem(int index) {
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

  // Build the whole list of todo items
  Widget _buildTodoList() {

    return ListView.builder(
      itemBuilder: (BuildContext _context, int index) {
        // itemBuilder will be automatically be called as many times as it takes for the
        // list to fill up its available space, which is most likely more than the
        // number of todo items we have. So, we need to check the index is OK.
        if(index < _todoItems.length) {
          return _buildTodoItem(_todoItems[index], index);
        }
      },
    );
  }

  Future<void> _getTodos() async {
    Map<String, dynamic> temp = await _replicant.exec('getTodos', []);
    List<dynamic> todos = List.from(temp.entries.map((e) { e.value['id'] = e.key; return e.value;} ));
    setState(() => _todoItems = todos);
  }

  // Build a single todo item
  Widget _buildTodoItem(var todoItem, int index) {
    return new ListTile(
      title: new Text(todoItem["title"]),
      onTap: () => _promptRemoveTodoItem(index)
      
    );
    
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
