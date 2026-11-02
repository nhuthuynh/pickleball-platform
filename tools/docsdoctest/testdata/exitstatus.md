# Exit status is not the claim

`grep` exits 1 when it matches nothing, and "matches nothing" is a perfectly
good documented claim. The gate compares output, not exit status.

```console doctest
$ printf 'present\n' | grep absent; printf 'still ran\n'
still ran
```
