class Todo {
  Todo.fromJson(String id, Map<String, dynamic> data)
      : id = id,
        title = data['title'],
        done = data['done'],
        order = data['order'] {
  }

  String id;
  String title;
  bool done;
  num order;
}
