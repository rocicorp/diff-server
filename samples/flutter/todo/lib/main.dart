import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

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
  static const platform = const MethodChannel('replicant.dev/samples/todo');

  _MyHomePageState() {
    _init();
  }

  int _counter = 0;

  Future<void> _init() async {
    await _registerBundle();
    await _refreshCounter();
  }

  Future<void> _incrementCounter() async {
    await platform.invokeMethod('exec', jsonEncode({'name': 'add', 'args': ['counter', 1]}));
    await _refreshCounter();
  }

  Future<void> _refreshCounter() async {
    Map<String, dynamic> resp = jsonDecode(await platform.invokeMethod('get', jsonEncode({'key': 'counter'})));
    setState(() {
      _counter = resp['data'] != null ? resp['data'] : 0;
    });
  }

  Future<void> _registerBundle() async {
    var bundle = await rootBundle.loadString('assets/bundle.js', cache: false);

    var bundleVersion = new RegExp(r"function codeVersion\(\) {[\n\s]+return (\d+);", multiLine: true).firstMatch(bundle);
    if (bundleVersion == null) {
      throw new Exception("Could not find codeVersion from bundle.");
    }

    var storedVersion = 0;
    /*
    TODO:aa
    try {
      String res = await platform.invokeMethod("exec", jsonEncode({"name": "codeVersion", "args": []}));
      print(res);
    } catch (e) {
      print("Error: " + e.toString());
    }
    */

    if (storedVersion < int.parse(bundleVersion.group(1))) {
      platform.invokeMethod("putBundle", jsonEncode({
        'code': bundle,
      }));
    }

    print("Replicant: Bundle registered");
  }

  Future<void> _sync() async {
    print("Syncing...");
    await platform.invokeMethod("sync", jsonEncode({
      'remote': 'https://replicate.to/serve/susan-counter',
    }));
    await _refreshCounter();
    print("Done");
  }

  Future <void> _dropDatabase() async {
    await platform.invokeMethod("dropDatabase");
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
    /*
    return new ListView.builder(
      itemBuilder: (context, index) {
        // itemBuilder will be automatically be called as many times as it takes for the
        // list to fill up its available space, which is most likely more than the
        // number of todo items we have. So, we need to check the index is OK.
        if(index < _todoItems.length) {
          return _buildTodoItem(_todoItems[index], index);
        }
      },
    );
    */
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
