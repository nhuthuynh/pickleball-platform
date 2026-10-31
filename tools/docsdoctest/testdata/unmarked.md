# Unmarked blocks

An illustrative block: the server is not running, so this cannot be executed.

```
$ curl -H "Authorization: Bearer $TOKEN" localhost:8080/v1/bookings
{"bookings": []}
```

A block with no prompt at all is not even a candidate — it is a tree, a
snippet, or a table:

```
proto/                     API contract
internal/<context>/
```
