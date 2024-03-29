## 2.0.3
  * improve internal metrics some
    * Track dropped tags and processing queue length
  * increase internal queue depth
    * For higher through-put situations, 10k vs 1k queue depth may matter.

## 2.0.2
  * handle bad metric type gracefully

## 2.0.1
  * Simplify passing variables in processor object
  * track dropped metrics during http request, too

## 2.0
  * Channel for processing metrics
    * significantly improve client performance (since client doesn't have to wait for entire processing pipelien to finish)
  * move batch writes to separate endpoint
    * allows for individual stats in a batch to be different types

## 1.7
  * batch writes into the `/:type` endpoint
    * many metrics in one write to the proxy, vastly improving performance over the wire
  * Go 1.19

## 1.6
  * prometheus-compat filtering for incoming metrics
  * fix metric prefix settings

## 1.5.2
  * bug with empty tag list fixed

## 1.5.1
  * allow for metric names in URLs _or_ in body

## 1.5
  * middleware to get better prometheus stat generation (RED/etc.)
  * Golang 1.18.5

## 1.4
  * metric names in message body
  * `/metrics` endpoint for prometheus-style stats tracking of proxy

## 1.3.1
  * Vendor updates

## 1.3
  * Go 1.18

## 1.2
  * Bump vendoring for jwt (fixes bugs in implementation)
  * merged changes from multiple projects

## 1.1
  * pull vendoring into local repo
  * build against Go 1.17
  * better linting/testing in build
  * pull in gometric statsd client
  * make JSON input values consistent (no `n` vs. `value` vs. `dur`) so clients can more easily send data

## 1.0-stable
  * use `alexcesaro/statsd`
  * adapt the stats interface

## 1.0-dev
  * Default HTTP port changed to 8825
  * Configure HTTP timeouts through `--http-timeout-read`, `--http-timeout-write` and `--http-timeout-idle`
  * Minumum required Go version is 1.8

## 0.9 (2019-01-29)
  * TLS Secure connection listening

## 0.8 (2018-02-24)
  * Added support of preflight CORS OPTIONS requests with header `X-JWT-Token`
  * Added support of authentication with passing token in query string instead of header `X-JWT-Token`

## 0.7 (2017-12-25)
  * Binary renamed to `statsd-http-proxy`
