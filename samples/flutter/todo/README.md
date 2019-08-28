# Repldo (Flutter)

A simple TODO app that demonstrates using Replicant from Flutter.

# Prerequisites

* Download and setup the [Flutter SDK](https://flutter.dev/docs/get-started/install)
* Get the repm (Replicant mobile) libraries, by either:
  * (Preferred, at the moment) Building them: See [repm/README.md](../../../repm/README.md)
  * [Downloading a prebuilt release](https://github.com/aboodman/replicant/releases)
    * Warning: We don't have the releases being automatically built right now, so it is highly likely to be out of date
    * Copy the repm.aar and Repm.framework.zip files into the `replicant/repm` directory of this project
    * Unzip the Repm.framework.zip file

We will eventually just build a Dart package that hides all of this.

# Build / Run

Once you have the prereqs, this is just a normal Flutter project, so:

```
flutter run
```

# Notes

## Nuking all the data

* If you just delete the local data and restart the app, the client will pull the server state.
* If you delete the server db and restart the app, the client will push its state to the server.
* Yay, sync!

To delete everything and start over, do this:

* Delete the app from all devices that are sharing the same server-side database
* From a command-line: `noms ds -d https://replicate.to/serve/<db-name>::local` (requires the Noms CLI)
