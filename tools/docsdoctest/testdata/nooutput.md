# A command whose claim is that it prints nothing

CLAUDE.md makes exactly this claim about `gate-coverage` and integration
files, with the same trailing-comment idiom.

```console doctest
$ printf 'x\n' | grep -i "nothing matches this"     # no output
```
