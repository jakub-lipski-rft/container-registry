# Medium-Scale Garbage Collection Testing

This guide provides instructions on setting up a registry in order to test
garbage collection. Of particular interest is seeding the registry with data and
preserving that data so that multiple successive garbage collection runs can be
as similar as possible to each other.

### Configuration

These configurations are minimal starting points to enable garbage
collection to run smoothly, though some relevant additional configuration
will be applied when it is impactful to garbage collection.

#### S3

This configuration assumes that you have access to an S3 bucket and can provide
an appropriate `accesskey` and `secretkey`. A setting of particular interest
to us is `maxrequestspersecond` which will increase or decrease the rate of
requests to S3. This setting will impact the time the mark state takes to
complete significantly.

```yaml
version: 0.1
log:
  fields:
    service: registry
storage:
  delete:
    enabled: true
  cache:
    blobdescriptor: inmemory
  s3:
    region: "us-east-1"
    bucket: "registry-bucket"
    encrypt: false
    secure: false
    maxrequestspersecond: 500
    accesskey: "<ACCESS_KEY>"
    secretkey: "<SECRET_KEY>"
  maintenance:
      uploadpurging:
          enabled: false
http:
  addr: :5000
  headers:
    X-Content-Type-Options: [nosniff]
```

## Seeding

After configuring and deploying the registry, we need to push images to it to
seed data for the garbage collection process.

First, we need to create a Dockerfile that will enable us to generate unique
layers that will not be shared across all images, without needing to alter the
contents of the Dockerfile itself.

```dockerfile
FROM alpine:3.11.3

RUN mkdir -p /root/test/
RUN head -c 524288 </dev/urandom > /root/test/randfile.txt
RUN date '+%s' > /root/test/date.txt
```

Now we can run a simple bash loop in the same directory as our
Dockerfile to seed 1201 images, which should produce around 6000 blobs. If you
are not running the container registry locally, you will need to replace
`localhost:5000` with the IP address of your container registry. Please note
this operation will take a considerable amount of time.

```bash
for i in {0..1200}; do
  docker build -t localhost:5000/test/alpine:${i} -f Dockerfile . \
  && docker push localhost:5000/test/alpine:${i} \
  && docker rmi localhost:5000/test/alpine:${i}
done
```

In order to test both the mark and sweep stage, we need to create a further
layers and remove any references to them. The simplest way to do this is to run
the loop again, specifying a different repository and then afterwards removing
that repository directly with the storage backend. This will remove references
to the layers uploaded to the repository, allowing them to be garbage collected,
if they are not referenced by other repositories.

```bash
for i in {0..1200}; do
  docker build -t localhost:5000/remove/alpine:${i} -f Dockerfile . \
  && docker push localhost:5000/remove/alpine:${i} \
  && docker rmi localhost:5000/remove/alpine:${i}
done
```

After removing the repository, you may wish to create a backup of the registry
storage in order to restore it. This enables testing garbage collection with
consistent data and is somewhat faster than repeating the seeding process.

Storage backend specific instructions on how to remove the repository and
perform backups and restores follow.

### S3

We will be working with the `s3cmd` utility to manage the S3 bucket directly.
Please ensure that you have this utility installed and configured such that you
have access to the S3 bucket your container registry is using. This utility
and installation and usage instructions can be found at https://s3tools.org/s3cmd

Substitute `registry-bucket` with the name the S3 bucket you are using in all
the following commands.

#### Deleting the Remove Repository

This command will delete the `remove` repository and all the files within:

```bash
s3cmd rm --recursive s3://registry-bucket/docker/registry/v2/repositories/remove/
```

#### Backing up and Restoring the Registry

These commands employ the `sync` subcommand to avoid unnecessary copying of
files via the `--skip--existing` flag.

Backup:
```bash
s3cmd sync --recursive --skip-existing s3://registry-bucket/docker/ s3://registry-bucket/backup/registry-1/
```

Restore:
```bash
s3cmd sync --recursive --skip-existing s3://registry-bucket/backup/registry-1/registry/ s3://registry-bucket/docker/
```


## Results

Running the garbage collection command (optionally with the `--dry-run` flag)
will produce a significant amount of of log entries. It might be useful to
redirect this output to a file in order to analyze the output.

```bash
docker exec -it walker /bin/registry garbage-collect --dry-run /etc/docker/registry/config.yml &> /path/to/file
```

Log entries are in a structured format which should allow for easy analysis.

All blobs, both in use and eligible for deletion, will be logged.

Entry from a blob in use (marked):
```
time="2020-01-31T17:43:02.81636132Z" level=info msg="marking manifest" digest=sha256:45d437916d5781043432f2d72608049dcf74ddbd27daa01a25fa63c8f1b9adc4 digest_type=layer go.version=go1.11.13 instance.id=1ba7d474-5018-4845-a1f6-871caebc8671 repository=ubuntu service=registry
```

Entry from a blob eligible for deletion (not marked):
```
time="2020-01-31T17:44:31.620341601Z" level=info msg="blob eligible for deletion" digest=sha256:b2214af1aed703acc22b29e146875e9cf3979f7c3a4f70e3c09c33ce174b6053 go.version=go1.11.13 instance.id=1ba7d474-5018-4845-a1f6-871caebc8671 path="/docker/registry/v2/blobs/sha256/b2/b2214af1aed703acc22b29e146875e9cf3979f7c3a4f70e3c09c33ce174b6053/data" service=registry
```

Of particular interest are the following entries, which mark the end of the
mark and sweep stages, respectively.

```
time="2020-01-31T17:44:31.61267147Z" level=info msg="mark stage complete" blobs_marked=15041 blobs_to_delete=9030 duration_s=91.933066704 go.version=go1.11.13 instance.id=1ba7d474-5018-4845-a1f6-871caebc8671 manifests_to_delete=0 service=registry
```

```
time="2020-01-31T17:44:34.851525753Z" level=info msg="sweep stage complete" duration_s=3.238775081 go.version=go1.11.13 instance.id=1ba7d474-5018-4845-a1f6-871caebc8671 service=registry
```
