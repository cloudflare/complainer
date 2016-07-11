# Complainer

Complainer's job is to send notifications to different services when tasks
fail on Mesos cluster. While your system should be reliable to failures of
individual tasks, it's nice to know when things fail and why.

Supported log upload services:

* No-op - keeps URLs to Mesos slave sandbox.
* S3 - both AWS S3 and on-premise S3-compatible API.

Supported reporting services:

* [Sentry](https://getsentry.com/) - a great crash reporting software.
* [Hipchat](https://www.hipchat.com/) - not so great communication platform.
* [Slack](https://slack.com/) - another communication platform.
* File - regular file stream output, including stdout/stderr.

## Quick start

Start sending all failures to Sentry:

```
docker run -it --rm cloudflare/complainer \
  -masters=http://mesos.master:5050 \
  -uploader=noop \
  -reporters=sentry \
  -sentry.dsn=https://foo:bar@sentry.dsn.here/8
```

Run this on Mesos itself!

![Sentry screenshot](screenshots/sentry.png)

## Reporting configuration

Complainer needs two command line flags to configure itself:

* `name` - Complainer instance name (default is `default`).
* `default` - Whether to use `default` instance for each reporter implicitly.
* `masters` - Mesos master URL list (ex: `http://host:port,http://host:port`).
* `listen` - Listen address for HTTP (ex: `127.0.0.1:8888`).

These settings can be applied by env vars as well:

* `COMPLAINER_NAME` - Complainer instance name (default is `default`).
* `COMPLAINER_DEFAULT` - Whether to use `default` instance for each reporter implicitly.
* `COMPLAINER_MASTERS` - Mesos master URL list (ex: `http://host:port,http://host:port`).
* `COMPLAINER_LISTEN` - Listen address for HTTP (ex: `127.0.0.1:8888`).

### HTTP interface

Complainer provides HTTP interface. You have to enable it with `-listen`
command line flag or with `COMPLAINER_LISTEN` env variable.

This interface is used for the following:

* Health checks
* [pprof](https://golang.org/pkg/net/http/pprof/) endpoint

#### Health checks

`/health` endpoint reports `200 OK` when things are operating mostly normally
and `500 Internal Server Error` when complainer cannot talk to Mesos.

We don't check for other issues (uploader and reporter failures) because they
are not guaranteed to be happening continuously to recover themselves.

#### pprof endpoint

`/debug/pprof` endpoint exposes the regular `net/http/pprof` interface:

* https://golang.org/pkg/net/http/pprof/

### Log upload services

Log upload service is specified by command line flag `uploader`.
Alternatively you can specify this by env var `COMPLAINER_UPLOADER`.
Only one uploader can be specified per complainer instance.

#### no-op

Uploader name: `noop`

No-op uploader just echoes Mesos slave sandbox URLs.

#### S3 AWS

Uploader name: `s3aws`.

This uploader uses official AWS SDK and should be used if you use AWS.

Stdout and stderr logs get uploaded to S3 and signed URLs provided to reporters.
Logs are uploaded into the following directory structure by default:

* `${YYYY-MM-DD}/complainer/${task_name}/${YYYY-MM-DDTHH:mm:ssZ}-${task_id}/{stdout,stderr}`

Command line flags:

* `s3aws.access_key` - S3 access key.
* `s3aws.secret_key` - S3 secret key.
* `s3aws.region` - S3 region.
* `s3aws.bucket` - S3 bucket name.
* `s3aws.prefix` - S3 prefix template (`Failure` struct is available).
* `s3aws.timeout` - Timeout for signed S3 URLs (ex: `72h`).

You can set value of any command line flag via environment variable. Example:

* Flag `s3aws.access_key` becomes env variable `S3_ACCESS_KEY`

Flags override env variables if both are supplied.

The minimum AWS policy for complainer is `s3:PutObject`:

* https://docs.aws.amazon.com/AmazonS3/latest/dev/using-with-s3-actions.html

##### S3 Compatible APIs

Uploader name: `s3goamz`.

This uploader uses goamz package and supports S3 compatible APIs that use
v2 style signatures. This includes Ceph Rados Gateway.

Stdout and stderr logs get uploaded to S3 and signed URLs provided to reporters.
Logs are uploaded into the following directory structure by default:

* `${YYYY-MM-DD}/complainer/${task_name}/${YYYY-MM-DDTHH:mm:ssZ}-${task_id}/{stdout,stderr}`

* `s3goamz.access_key` - S3 access key.
* `s3goamz.secret_key` - S3 secret key.
* `s3goamz.endpoint` - S3 endpoint (ex: `https://complainer.s3.example.com`).
* `s3goamz.bucket` - S3 bucket name.
* `s3goamz.prefix` - S3 prefix template (`Failure` struct is available).
* `s3goamz.timeout` - Timeout for signed S3 URLs (ex: `72h`).

You can set value of any command line flag via environment variable. Example:

* Flag `s3goamz.access_key` becomes env variable `S3_ACCESS_KEY`

Flags override env variables if both are supplied.

### Reporting services

Reporting services are specified by command line flag `reporters`.
Alternatively you can specify this by env var `COMPLAINER_REPORTERS`.
Several services can be specified, separated by comma.

#### Sentry

Command line flags:

* `sentry.dsn` - Default Sentry DSN to use for reporting.

Labels:

* `dsn` - Sentry DSN to use for reporting.

If label is unspecified, command line flag value is used.

#### Hipchat

Command line flags:

* `hipchat.base_url` - Base Hipchat URL, needed for on-premise installations.
* `hipchat.room` - Default Hipchat room ID to send notifications to.
* `hipchat.token` - Default Hipchat token to authorize requests.
* `hipchat.format` - Template to use in messages.

Labels:

* `base_url` - Hipchat URL, needed for on-premise installations.
* `room` - Hipchat room ID to send notifications to.
* `token` - Hipchat token to authorize requests.

If label is unspecified, command line flag value is used.

Templates are based on [`text/template`](https://golang.org/pkg/text/template/).
The following fields are available:

* `failure` - Failure struct.
* `stdoutURL` - URL of the stdout stream.
* `stderrURL` - URL of the stderr stream.

#### Slack

Command line flags:

* `slack.hook_url` - Webhook URL, needed to post something (required).
* `slack.channel` - Channel to post into, e.g. #mesos (optional).
* `slack.username` - Username to post with, e.g. "Mesos Cluster" (optional).
* `slack.icon_emoji` - Icon Emoji to post with, e.g. ":mesos:" (optional).
* `slack.icon_url` - Icon URL to post with, e.g. "http://my.com/pic.png" (optional).
* `slack.format` - Template to use in messages.

Labels:

* `hook_url` - Webhook URL, needed to post something (required).
* `channel` - Channel to post into, e.g. #mesos (optional).
* `username` - Username to post with, e.g. "Mesos Cluster" (optional).
* `icon_emoji` - Icon Emoji to post with, e.g. ":mesos:" (optional).
* `icon_url` - Icon URL to post with, e.g. "http://my.com/avatar.png" (optional).

If label is unspecified, command line flag value is used.

For more details see [Slack API docs](https://api.slack.com/incoming-webhooks).

Templates are based on [`text/template`](https://golang.org/pkg/text/template/).
The following fields are available:

* `failure` - Failure struct.
* `stdoutURL` - URL of the stdout stream.
* `stderrURL` - URL of the stderr stream.

#### File

Command line flags:

* `file.name` - File name to output logs.
* `file.format` - Template to use in output logs.

Templates are based on [`text/template`](https://golang.org/pkg/text/template/).
The following fields are available:

* `failure` - Failure struct.
* `stdoutURL` - URL of the stdout stream.
* `stderrURL` - URL of the stderr stream.

### Label configuration

#### Basics

To support flexible notification system, Mesos task labels are used. Marathon
task labels get copied to Mesos labels, so these are equivalent.

The minimal set of labels needed is an empty set. You can configure default
values in Complainer's command line flags and get all notifications with
these settings. In practice, you might want to have different reporters for
different apps.

Full format for complainer label name looks like this:

* `complainer_${name}_${reporter}_instance_${instance}_${key}`

Example (`dsn` set for `default` Sentry of `default` Complainer):

* `complainer_default_sentry_instance_default_dsn`

This is long and complex, so default parts can be skipped:

* `complainer_sentry_dsn`

#### Advanced labels

The reason for having long label name version is to add the flexibility.
Imagine you want to report app failures to the internal Sentry, two internal
Hipchat rooms (default and project-specific) and the external Sentry.

Set of labels would look like this:

* `complainer_sentry_dsn: ABC` - for internal Sentry.
* `complainer_hipchat_instances: default,myapp` - adding instance `myapp`.
* `complainer_hipchat_instance_myapp_room: 123`- setting room for `myapp`.
* `complainer_hipchat_instance_myapp_token: XYZ`- setting token for `myapp`.
* `complainer_external_sentry_dsn: FOO` - for external Sentry.

Internal and external complainers can have different upload services.

Implicit instances are different, depending on how you run Complainer.

* `-default=true` (default) - `default` instance is implicit.
* `-default=false` - no instances are configured implicitly.

The latter is useful for opt-in monitoring, including monitoring of Complainer
itself (also known as dogfooding).

#### Templating

Templates are based on [`text/template`](https://golang.org/pkg/text/template/).
The following fields are available:

* `nl` - Newline symbol (`\n`).
* `config` - Function to get labels for the reporter.
* `failure` - Failure struct: https://godoc.org/github.com/cloudflare/complainer#Failure
* `stdoutURL` - URL of the stdout stream.
* `stderrURL` - URL of the stderr stream.

With `config` you can use labels in templates. For example, the following
template for the Slack reporter:

```
Task {{ .failure.Name }} ({{ .failure.ID }}) died | {{ config "mentions" }}{{ .nl }}
```

With the label `complainer_slack_mentions=@devs` will be evaluated to:

```
Task foo.bar (bar.foo.123) died | @devs
```

#### Dogfooding

To report errors for complainer itself you need to run two instances:

* `default` to monitor all other tasks.
* `dogfood` to monitor the `default` Complainer.

You'll need the following labels for the `default` instance:

```yaml
labels:
  complainer_dogfood_sentry_instances: default
  complainer_dogfood_hipchat_instances: default
```

For the `dogfood` instance you'll need to:

* Add `-name=dogfood` command line flag.
* Add `-default=false` command line flag.

Since the `dogfood` Complainer ignores apps with not configured instances,
it will ignore every failure except for the `default` instance failures.

If the `dogfood` instance fails, `default` reports it just like any other task.

If both instances fail at the same time, you get nothing.

## Copyright

* Copyright 2016 CloudFlare

## License

MIT
