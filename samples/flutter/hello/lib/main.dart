import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:replicant/replicant.dart';

void main() => runApp(MyApp());

class MyApp extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Flutter Demo',
      theme: ThemeData(
        primarySwatch: Colors.blue,
      ),
      home: MyHomePage(title: 'Flutter Demo Home Page'),
    );
  }
}

class MyHomePage extends StatefulWidget {
  MyHomePage({Key key, this.title}) : super(key: key);

  final String title;

  @override
  _MyHomePageState createState() => _MyHomePageState();
}

class _MyHomePageState extends State<MyHomePage> {
  int _counter = 0;

  Replicant _replicant;

  _MyHomePageState() {
    this.init();
  }

  void init() async {
    _replicant = Replicant('https://replicate.to/serve/demo-counter');
    _replicant.onChange = this._handleChange;
    _replicant.onSync = this._handleSync;
    await _replicant.putBundle(await rootBundle.loadString('lib/bundle.js', cache: false));
    this._handleChange();
  }

  void _incrementCounter() {
    _replicant.exec('incr', [1]);
  }

  void _handleChange() async {
    var count = await _replicant.exec('getCount');
    setState(() {
      _counter = count;
    });
  }

  void _handleSync(bool syncing) {
    if (syncing) {
      print('Syncing...');
    } else {
      print('Done');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.title),
      ),
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: <Widget>[
            Text(
              'You have pushed the button this many times:',
            ),
            Text(
              '$_counter',
              style: Theme.of(context).textTheme.display1,
            ),
          ],
        ),
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: _incrementCounter,
        tooltip: 'Increment',
        child: Icon(Icons.add),
      ),
    );
  }
}
