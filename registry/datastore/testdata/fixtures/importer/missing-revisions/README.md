# Missing Revisions

This test fixture simulates a registry with a repository in which the revisions
path is missing. Malformed registies such as these can appear when an
administrator performs manual maintaince against filesystem metadata.

## Fixture Creation

This fixure was created by uploading two schema 2 images and removing the
`_manifests/revisions/` path in one of them.
