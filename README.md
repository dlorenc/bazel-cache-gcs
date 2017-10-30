# A Bazel HTTP Cache Backed by GCS

This project acts as a bazel "distributed" cache that stores contents on GCS.
To read more about the bazel cache protocol, see the [documentation](https://github.com/bazelbuild/bazel/blob/master/src/main/java/com/google/devtools/build/lib/remote/README.md).

## Usage

Configure bazel to use a cache by passing something like this in your .bazelrc:

```shell
startup --host_jvm_args=-Dbazel.DigestFunction=SHA1
build --spawn_strategy=remote
build --remote_rest_cache=http://localhost:8080/
# Bazel currently doesn't support remote caching in combination with workers.
# These two lines override the default strategy for Javac and Closure
# actions, so that they are also remotely cached.
# Follow https://github.com/bazelbuild/bazel/issues/3027 for more details:
build --strategy=Javac=remote
build --strategy=Closure=remote
```

Then, build and start the cache with bazel:

```shell
bazel run //:bazel-cache-gcs -- --bucket=$SOME_BUCKET
```

The server listens on localhost:8080 by default, this can be overridden with the "--port" flag.

This project uses the standard google-cloud-go client libraries to interact with GCS, so any of the
authentication methods described in the documentation should work out of the box.

Note that this can be enabled/disabled in your project root or in your $HOME directory.
In this project, it is explictly disabled because bazel fails when the cache is not running.
This makes it easier to build the cache itself :)

## Protocol

The documentation on the HTTP API appears to be missing, so I reverse engineered it a bit.
It appears bazel sends GET and PUT requests to the configured server, with a content-addressable URL path.

GET requests are meant to be check the cache, and PUT requests add new values to it.

## GCS Naming

This project simply turns the bazel URL paths directly into GCS object keys, in a bucket named at startup.
If the object exists, we return the contents in the HTTP response body.
If the object does not exist, we 404.
