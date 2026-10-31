# Marked blocks

A paragraph claiming something measurable.

```console doctest
$ printf 'one\ntwo\n'
one
two
```

Indented under a list item, which CommonMark allows and this repo's own
CLAUDE.md uses for every block in its Gotchas section:

- a bullet

  ```console doctest
  $ printf 'indented\n'
  indented
  ```

A marked block holding two commands, separated by a blank line — the blank
belongs to the first command's expected output and must be trimmed away:

```console doctest
$ printf 'first\n'
first

$ printf 'second\n'
second
```
