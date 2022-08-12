# StatsD Rest Proxy

*Note: This repository is a combination of forks from https://github.com/InjectiveLabs/statsd-http-proxy and https://github.com/GoMetric/statsd-http-proxy. Along with an improved build story.*

StatsD uses UDP connections, and  can not be used directly from browser. This server is a HTTP proxy to StatsD, useful for sending metrics to StatsD from frontend by AJAX.

Requests may be optionally authenticated using JWT tokens.

## Table of contents

* [Installation](#installation)
* [Nginx config](#nginx-config)
* [Usage](#usage)
* [Client Interactions](#client-interactions)

## Installation

```
# clone repo
make
```

## Nginx config

Configuration of Nginx balancer:

```config
server {
    listen 443 http2;
    server_name statsd-proxy.example.com;
    ssl on;
    ssl_certificate     /etc/pki/nginx/ssl.crt;
    ssl_certificate_key /etc/pki/nginx/ssl.key;
    upstream statsd_proxy {
        keepalive 100;
        server statsd-proxy-1:8825 max_fails=0;
        server statsd-proxy-2:8825 max_fails=0;
    }

    location / {
        proxy_pass http://statsd_proxy;
        proxy_redirect off;
        proxy_http_version 1.1;
        proxy_set_header Connection "keep-alive";
    }
}
```

## Caddy config

```config
localhost {
    handle /statsd* {
        uri strip_prefix /statsd
        # suppress external access to /metrics and /heartbeat endpoints
        redir /metrics / permanent
        redir /heartbeat / permanent
        reverse_proxy statsd-proxy:80
    }
}
```

## Usage

* Run server (HTTP):

```bash
statsd-http-proxy \
    --verbose \
    --http-host=127.0.0.1 \
    --http-port=8080 \
    --statsd-host=127.0.0.1 \
    --statsd-port=8125 \
    --jwt-secret=somesecret \
    --metric-prefix=prefix.subprefix
```

* Run server (HTTPS):

```bash
statsd-http-proxy \
    --verbose \
    --http-host=127.0.0.1 \
    --http-port=433 \
    --tls-cert=cert.pem \
    --tls-key=key.pem \
    --statsd-host=127.0.0.1 \
    --statsd-port=8125 \
    --jwt-secret=somesecret \
    --metric-prefix=prefix.subprefix
```

Print server version and exit:

```bash
statsd-http-proxy --version
```

Command line arguments:

| Parameter       | Description                          | Default value                                                                     |
|-----------------|--------------------------------------|-----------------------------------------------------------------------------------|
| verbose         | Print debug info to stderr           | Optional. Default false                                                           |
| http-host       | Host of HTTP server                  | Optional. Default 127.0.0.1. To accept connections on any interface, set to ""    |
| http-port       | Port of HTTP server                  | Optional. Default 80                                                              |
| http-timeout-read | The maximum duration in seconds for reading the entire request, including the body | Optional. Defaults to 1 second |
| http-timeout-write | The maximum duration in seconds before timing out writes of the respons | Optional. Defaults to 1 second  |
| http-timeout-idle | The maximum amount of time in seconds to wait for the next request when keep-alives are enabled | Optional. Defaults to 1 second |
| tls-cert        | TLS certificate for the HTTPS        | Optional. Default "" to use HTTP. If both tls-cert and tls-key set, HTTPS is used |
| tls-key         | TLS private key for the HTTPS        | Optional. Default "" to use HTTP. If both tls-cert and tls-key set, HTTPS is used |
| statsd-host     | Host of StatsD instance              | Optional. Default 127.0.0.1                                                       |
| statsd-port     | Port of StatsD instance              | Optional. Default 8125                                                            |
| jwt-secret      | JWT token secret                     | Optional. If not set, server accepts all connections                              |
| metric-prefix   | Prefix, added to any metric name     | Optional. If not set, do not add prefix                                           |
| version         | Print version of server and exit     | Optional                                                                          |

## Client Interactions

Sample code to send metric in browser with JWT token in header:

```javascript
$.ajax({
    url: 'http://127.0.0.1:8080/count/',
    method: 'POST',
    headers: {
        'X-JWT-Token': 'some-jwt-token'
        'Content-Type': 'application/json'
    },
    data: {
        metric: 'some.key.name',
        value: 100500
    }
});
```
## Supported metrics

For the general reference see https://www.librato.com/docs/kb/collect/collection_agents/stastd/#

All metrics accept `tags` as comma-separated key=value pairs (InfluxDB tag format):

```javascript
data: {
    metric: 'some.key.name',
    value: 100500,
    tags: 'env=prod,locale=en-us'
}
```

### `count`

Adds count to the bucket. Expected `value` as integer. By default `value` is 0.

### `gauge`

Sets the gauge metric. Expected `value` as integer. Before setting negative gauge, it needs to be set to `0`.

### `timing`

Adds timing to the bucket. Expected `value` as milliseconds integer. Default is `0`.

### `set`

Adds value in a set bucket. Expected `value` as string. Sets are a relatively new concept in recent versions of StatsD. Sets track the number of unique elements belonging to a group. At each flush interval, the statsd backend will push the number of unique elements in the set as a single gauge value.

## Legacy Pattern

You can also send metrics in using the legacy pattern:

```javascript
$.ajax({
    url: 'http://127.0.0.1:8080/count/some.key.name',
    method: 'POST',
    headers: {
        'X-JWT-Token': 'some-jwt-token'
        'Content-Type': 'application/json'
    },
    data: {
        value: 100500
    }
});
```

We define the metric name in the URL rather than the data object. This _does_ slightly restrict the allowed characters a metric name can have (because allowed URL characters are more limited).

All other suggestions/requirements from above apply to metrics generated this way.
