/// DatabaseInfo contains information about available local databases.
class DatabaseInfo {
  DatabaseInfo.fromJSON(Map<String, dynamic> data)
    : name = data['name'] {
    }
  String name;
}