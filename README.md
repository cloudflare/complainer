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

## Configuration parameters

Complainer's runtime configuration can be passed using command line options and via environment variables.

Values passed on the command line override those in environment variables. All command line options expect 
a parameter, unless otherwise specified. Valid syntaxes for parametrized options are `-option=parameter` and
`-option parameter`.


| Cmdline      | Environment            |   | Description                                                              |
|--------------|------------------------|---|--------------------------------------------------------------------------|
| `-masters`   | `COMPLAINER_MASTERS`   | * | Mesos master URL list in the form `http://host:port,http://host:port,...` |
| `-uploader`  | `COMPLAINER_UPLOADER`  | * | Upload service to enable. One of `noop`, `s3aws`, `s3goamz` |
| `-reporters` | `COMPLAINER_REPORTERS` | * | Comma-separated list of reporter services to enable |
| `-name`      | `COMPLAINER_NAME`      | | Complainer instance name (default: `default`) |
| `-default`   | `COMPLAINER_DEFAULT`   | ? | Whether to use `default` instance for each reporter implicitly |
| `-listen`    | `COMPLAINER_LISTEN`    | | Listen address for HTTP (example: `127.0.0.1:8888`) |
|              | `PORT`                 | | If `-listen`/`COMPLAINER_LISTEN` is not defined, but `PORT` is, complainer will start the HTTP listener on the port `PORT`. This value must be a plain integer without the preceding colon |
| `-framework-whitelist` | | | List of regexps. Only frameworks whose names match the list are reported |
| `-framework-blacklist` | | | List of regexps. Frameworks whose names match the list are not reported |
||||||
| `-s3aws.access_key`     | `S3_ACCESS_KEY` | | S3 access key |
| `-s3aws.secret_key`     | `S3_SECRET_KEY` | | S3 secret key |
| `-s3aws.region`         | `S3_REGION`     | | S3 region |
| `-s3aws.bucket`         | `S3_BUCKET`     | | S3 bucket name |
| `-s3aws.prefix`         | `S3_PREFIX`     | | S3 prefix template. "`Failure`" struct is available |
| `-s3aws.timeout`        | `S3_TIMEOUT`    | | Timeout for signed S3 URLs (ex: `72h`). Default: 7 days |
||||||
| `-s3goamz.access_key`   | `S3_ACCESS_KEY` | | S3 access key|
| `-s3goamz.secret_key`   | `S3_SECRET_KEY` | | S3 secret key|
| `-s3goamz.endpoint`     | `S3_ENDPOINT`   | | S3 endpoint (ex: `https://complainer.s3.example.com`)|
| `-s3goamz.bucket`       | `S3_BUCKET`     | | S3 bucket name|
| `-s3goamz.prefix`       | `S3_PREFIX`     | | S3 prefix template. `Failure` struct is available  |
| `-s3goamz.timeout`      | `S3_TIMEOUT`    | | Timeout for signed S3 URLs (ex: `72h`)|
||||||
| `-sentry.dsn`           | `SENTRY_DSN`    | | Sentry DSN. Default: "" |
||||||
| `-hipchat.base_url`     | `HIPCHAT_BASE_URL` | |  Base Hipchat URL, needed for on-premise installations. Default: "`https://api.hipchat.com/v2/`" |
| `-hipchat.room`         | `HIPCHAT_ROOM`     | |  Default Hipchat room ID to send notifications to |
| `-hipchat.token`        | `HIPCHAT_TOKEN`    | |  Default Hipchat token to authorize requests |
| `-hipchat.format`       | `HIPCHAT_FORMAT`   | |  Template to use in messages. |
||||||
| `-slack.hook_url`       | `SLACK_HOOK_URL`   | | Webhook URL, needed to post something (required) |
| `-slack.channel`        | `SLACK_CHANNEL`    | | Channel to post into, e.g. #mesos (optional) |
| `-slack.username`       | `SLACK_USERNAME`   | | Username to post with, e.g. "Mesos Cluster" (optional) |
| `-slack.icon_emoji`     | `SLACK_ICON_EMOJI` | | Icon Emoji to post with, e.g. ":mesos:" (optional) |
| `-slack.icon_url`       | `SLACK_ICON_URL`   | | Icon URL to post with, e.g. "http://my.com/pic.png" (optional) |
| `-slack.format`         | `SLACK_FORMAT`     | | Template to use in messages |
||||||
| `-jira.url`             | `JIRA_URL`         | | Default JIRA instance url (required) |
| `-jira.username`        | `JIRA_USERNAME`    | | JIRA user to authenticate as (required) |
| `-jira.password`        | `JIRA_PASSWORD`    | | JIRA password for the user to authenticate (required) |
| `-jira.issue_closed_status` | `JIRA_ISSUE_CLOSED_STATUS` | | The status of JIRA issue when it is considered closed |
| `-jira.fields`          | `JIRA_FIELDS` | | JIRA fields, see below |
||||||
| `-file.name`            | `FILE_NAME`    | | File name to output logs. Default: /dev/stderr |
| `-file.format`          | `FILE_FORMAT`  | | Log format |

\* = Mandatory option

? = Boolean flag, does not expect a parameter

## Filtering based on the failure's framework

If you're in a situation where you have multiple marathons running against
a mesos, and want to segregate out which failures go where, the following
options are of interest. Each option can be specified multiple times.

* `framework-whitelist` - This is a regex option; if given, the failures
  framework must match at least one whitelist. If no whitelist is specified,
  then it's treated as if '.*' had been passed- all failures are whitelisted
  as long as they don't match a blacklist.
* `framework-blacklist` - This is a regex option; if given, any failure that
  matches this are ignored.

Note that the order of evaluation is such that blacklists are applied first,
then whitelists.

## HTTP interface

Complainer provides an HTTP interface. You have to enable it using the
`-listen`/ `COMPLAINER_LISTEN` or `PORT` parameters.

This interface is used for the following:

* Health checks
* [pprof](https://golang.org/pkg/net/http/pprof/) endpoint

### Health checks

`/health` endpoint reports `200 OK` when things are operating mostly normally
and `500 Internal Server Error` when complainer cannot talk to Mesos.

We don't check for other issues (uploader and reporter failures) because they
are not guaranteed to be happening continuously to recover themselves.

### pprof endpoint

`/debug/pprof` endpoint exposes the regular `net/http/pprof` interface:

* https://golang.org/pkg/net/http/pprof/

## Log upload services

Log upload service is specified by the `uploader` /  `COMPLAINER_UPLOADER`
parameter. Only one uploader can be specified per complainer instance.

### No-op uploader

Uploader name: `noop`

No-op uploader just echoes Mesos slave sandbox URLs.

### S3 AWS uploader

Uploader name: `s3aws`.

This uploader uses official AWS SDK and should be used if you use AWS.

Stdout and stderr logs get uploaded to S3 and signed URLs provided to reporters.

Default value for `-s3aws.prefix` / `S3_PREFIX`:

`complainer/{{ .failure.Finished.UTC.Format \"2006-01-02\" }}/{{ .failure.Name }}/{{ .failure.Finished.UTC.Format \"2006-01-02T15:04:05.000\" }}-{{ .failure.ID }}`

The minimum AWS policy for complainer is `s3:PutObject`:

* https://docs.aws.amazon.com/AmazonS3/latest/dev/using-with-s3-actions.html

### S3 Compatible API uploader

Uploader name: `s3goamz`.

This uploader uses goamz package and supports S3 compatible APIs that use
v2 style signatures. This includes Ceph Rados Gateway.

Stdout and stderr logs get uploaded to S3 and signed URLs provided to reporters.

Default value for `-s3goamz.prefix` / `S3_PREFIX`:

`complainer/{{ .failure.Finished.UTC.Format \"2006-01-02\" }}/{{ .failure.Name }}/{{ .failure.Finished.UTC.Format \"2006-01-02T15:04:05.000\" }}-{{ .failure.ID }}`



## Reporting services

Reporting services are specified `reporters` / `COMPLAINER_REPORTERS`
parameter. Several services can be specified, separated by comma.

### Sentry reporter

* `sentry.dsn` - Default Sentry DSN to use for reporting.

Labels:

* `dsn` - Sentry DSN to use for reporting.

If label is unspecified, command line flag value is used.

### Hipchat reporter

Default value for `-hipchat.format` / `HIPCHAT_FORMAT`:

`Task {{ .failure.Name }} ({{ .failure.ID }}) died with status {{ .failure.State }} [<a href=\"{{ .stdoutURL }}\">stdout</a>, <a href=\"{{ .stderrURL }}\">stderr</a>]`

Labels:

* `base_url` - Hipchat URL, needed for on-premise installations.
* `room` - Hipchat room ID to send notifications to.
* `token` - Hipchat token to authorize requests.

If label is unspecified, command line flag value is used.

### Slack reporter

Labels:

* `hook_url` - Webhook URL, needed to post something (required).
* `channel` - Channel to post into, e.g. #mesos (optional).
* `username` - Username to post with, e.g. "Mesos Cluster" (optional).
* `icon_emoji` - Icon Emoji to post with, e.g. ":mesos:" (optional).
* `icon_url` - Icon URL to post with, e.g. "http://my.com/avatar.png" (optional).

If label is unspecified, command line flag value is used.

For more details see [Slack API docs](https://api.slack.com/incoming-webhooks).

### Jira reporter

The `-jira.fields` / `JIRA_FIELDS` parameter defines JIRA fields in `key:value;...` format separated by `;`. It must contain `Project`, `Summary` and `Issue Type`. Default:

`Project:COMPLAINER;Issue Type:Bug;Summary:Task {{ .failure.Name }} died with status {{ .failure.State }};Description:[stdout|{{ .stdoutURL }}], [stderr|{{ .stderrURL }}], ID={{ .failure.ID }}`

### File reporter

Default value for `-file.format` / `FILE_FORMAT`:

`"Task {{ .failure.Name }} ({{ .failure.ID }}) died with status {{ .failure.State }}:{{ .nl }}  * {{ .stdoutURL }}{{ .nl }}  * {{ .stderrURL }}{{ .nl }}"`

## Label configuration

### Basics

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

### Advanced labels

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

## Configuration templating

Various configuration parameters support evaluating their values dynamically
using Golang's [`text/template`](https://golang.org/pkg/text/template/) 
templating engine.

The following fields are available for creating templates:

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

## Dogfooding

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
