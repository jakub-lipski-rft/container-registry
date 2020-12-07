# Invalid Blob

This test fixture simulates a registry with an invalid blob data path spec. This
is intended to simulate a failure during blob enumeration.

## Fixture Creation

This fixure was created by truncating the blob data path of a single blob:

```
mv 5b/5bd0410a6c06bea7199267f377dd0610898410bd1bf63828343c32daea8f0da3/ 5b/5bd0
```

