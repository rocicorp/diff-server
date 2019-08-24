import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'model.dart';
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
        // This is the theme of your application.
        //
        // Try running your application with "flutter run". You'll see the
        // application has a blue toolbar. Then, without quitting the app, try
        // changing the primarySwatch below to Colors.green and then invoke
        // "hot reload" (press "r" in the console where you ran "flutter run",
        // or simply save your changes to "hot reload" in a Flutter IDE).
        // Notice that the counter didn't reset back to zero; the application
        // is not restarted.
        primarySwatch: Colors.blue,
      ),
      home: MyHomePage(title: 'Flutter Demo Home Page'),
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

  List<Todo> _todos;

  Future<void> _init() async {
    await _registerBundle();
    await _load();
  }

  Future<void> _load() async {
    final Map<String, dynamic> res = await _replicant.exec('getTodos');
    var todos = List.from(res.entries.map((e) => Todo.fromJson(e.key, e.value)));
    todos.sort((t1, t2) => t1.order < t2.order ? -1 : t1.order == t2.order ? 0 : 1);
    setState(() {
      _todos = todos;
    });
  }

  Future<void> _registerBundle() async {
    var registeredVersion = 0;
    try {
      registeredVersion = await _replicant.exec('codeVersion');
    } on Exception catch(e) {
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

  Future<void> _sync() async {
    print("Syncing...");
    await _replicant.sync('https://replicate.to/serve/susan-counter');
    await _load();
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
            child: Text("Hello!"),
              decoration: BoxDecoration(
              color: Colors.blue,
            ),
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

  void _addTodoItem(String task) {
    /*
    // Only add the task if the user actually entered something
    if(task.length > 0) {
      // Putting our code inside "setState" tells the app that our state has changed, and
      // it will automatically re-render the list
      setState(() => _todoItems.add(task));
    }
    */
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
    return new ListView.builder(
      itemBuilder: (context, index) {
        // itemBuilder will be automatically be called as many times as it takes for the
        // list to fill up its available space, which is most likely more than the
        // number of todo items we have. So, we need to check the index is OK.
        if(index < _todos.length) {
          return _buildTodoItem(_todos[index].title, index);
        }
      },
    );
  }

  // Build a single todo item
  Widget _buildTodoItem(String todoText, int index) {
    return new ListTile(
      title: new Text(todoText),
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
