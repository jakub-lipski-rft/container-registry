# Storage Driver Local Integration Testing Guide

## Using minio to test the S3 storage driver.

This section will use a conainerized minio server to simulate an S3 storage
server. This can be used to test the S3 storage driver locally.

### Setting up the environment

In a terminal, set up the following environment variables:

```bash
export AWS_REGION="us-east-2"
export REGION_ENDPOINT="http://127.0.0.1:9000"
export S3_ENCRYPT="false"
export AWS_SECRET_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
export AWS_ACCESS_KEY="AKIAIOSFODNN7EXAMPLE"
export S3_BUCKET="test"
 ```

Next, run the minio server:

```bash
docker run -d -p 9000:9000 --name s3-test-mino \
  -e "MINIO_ACCESS_KEY=$AWS_ACCESS_KEY" \
  -e "MINIO_SECRET_KEY=$AWS_SECRET_KEY" \
  minio/minio server /data
```

The following command will create the test bucket without having to install
the minio client locally:

```bash
docker run --network=host -t --entrypoint=/bin/sh minio/mc \
  -c "mc config host add mino $REGION_ENDPOINT $AWS_ACCESS_KEY $AWS_SECRET_KEY && \
  mc mb mino/$S3_BUCKET"
```

Now you can run the S3 integration tests against the minio server we created above:

```bash
go test -timeout 20m -v github.com/docker/distribution/registry/storage/driver/s3-aws -args -check.v
```

Finally, the minio server can be stopped once you are finished with the
integration tests:

```bash
docker stop s3-test-minio
```
