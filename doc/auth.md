# Authorization

The Replicant server can optionally limit access to authorized clients via signed [JWT](https://jwt.io)s.

## Creating a Key Pair

You will need an elliptic curve key pair to sign auth tokens:

```
openssl ecparam -name prime256v1 -genkey -noout -out key.pem
openssl ec -in key.pem -pubout -out key.pub
```

Send `key.pub` to me (aa@roci.dev) and keep `key.pem` private.

## Token Structure

Replicant currently supports two JWT fields:

* `db`: The Replicant database name the token grants access to
* `exp`: The standard JWT expires field, which is seconds since the unix epoch

## Signing Your Token

Your server will need to create tokens and send them to clients periodically. The method to sign varies by
language/environment. See [jwt.io](https://jwt.io/) to find a compatible implementation.

Replicant JWTs are signed with ES256.

## Sending Token to Replicant

Pass the signed JWT to the `Replicant` constructor in your SDK. You can also update it from time to time via the public accessor:

```dart
var rep = Replicant("https://replicate.to/serve/4/mydb", authToken);

... time passes, client refreshes auth token from server ...

rep.authToken = newAuthToken
```
