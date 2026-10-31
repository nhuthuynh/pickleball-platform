# Backslash continuations

CLAUDE.md's integration-test census is written this way, so an extractor that
read the continuation lines as expected output would mis-parse it completely.

```console doctest
$ for w in a b; do \
    printf '%s\n' "$w"; \
  done
a
b
```
