# Unlinked Configuration Blob

This test fixture simulates a registry with a repository in which a
manifest configuration is unlinked from the repository containing the manifest.
Registry admins have access to the
[delete blob](https://gitlab.com/gitlab-org/container-registry/-/blob/master/docs/spec/api.md#delete-blob)
endpoint, which will delete the linked for the specified blob in the repository.

## Fixture Creation

This fixure was created by uploading three schema 2 images and removing the link
for the manifest configuration blob of the last image.
```
rm repositories/c-unlinked-config-blob/_layers/sha256/e4dd2892a904472ee05805501a1f0ea3776b4a549a61d4fa47d924da1f3585b1/link
```

