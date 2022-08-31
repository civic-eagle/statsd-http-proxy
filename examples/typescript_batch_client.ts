/*
 * It's important to clarify that this utility writes to a HTTP Proxy
 * in front of a statsd instance, not directly to statsd.
 * We do this to ensure we can properly encrypt and secure traffic
 * to our statsd instance (which normally, given statsd leverages UDP, we can't do)
 */
import axios from 'axios';
import http from 'http';
import https from 'https';
import jwt from 'jsonwebtoken';
import _ from 'lodash';
import NodeCache from 'node-cache';

const enabled = true;
const batchSize = 100;
// turn seconds into milliseconds
const batchTiming = 15 * 1000;

// the endpoint won't change while the program is running, so just process this once
let endpoint = 'http://127.0.0.1/';
if (endpoint.endsWith('/')) {
  endpoint = endpoint.slice(0, endpoint.length - 1);
}
console.log(`Metrics will be sent to ${endpoint}`);
const jwtSecret = 'jwt-secret';
// defined in hours, the length of time a generated JWT token lives
const expiresIn = '1h';
// do our best to get an environment that isn't "undefined"
const environment = process.env.CE_ENV || process.env.NODE_ENV;
// we always add `environment` as a label so we can filter dev/prod easily
const defaultTags = `environment=${environment}`;
/*
 * set cache length to half the period of the JWT token lifetime
 * easy to do 'cause we know expiresIn is a number of hours
 */
const statsdCache = new NodeCache({ stdTTL: (expiresIn * 60 * 60) / 2 });
const tlsAgentConfig = { keepAlive: true };

let metricsBatch: {
  metric_type: string;
  tags: string;
  metric: string;
  value: number;
  sampleRate: number;
}[] = [];

let thisInstance;
if (endpoint && enabled) {
  thisInstance = axios.create({
    baseURL: endpoint,
    timeout: 1500,
    headers: { 'Content-Type': 'application/json' },
    // Keep-Alive settings for Axios objects
    httpAgent: new http.Agent({ keepAlive: true }),
    httpsAgent: new https.Agent(tlsAgentConfig),
  });
}

/*
 * Add the `force` setting to ensure
 * we at least _attempt_ to send.
 * Useful when we're running a scheduled
 * send vs. a regular batch send
 */
const sendMetrics = async (force: boolean): Promise<void> => {
  if (!endpoint || !enabled) return;

  if ((force && metricsBatch.length > 0) || metricsBatch.length > batchSize) {
    /*
     * Only check for token expiration when we're about to send
     */
    let token = statsdCache.get('token');
    if (!token) {
      token = jwt.sign({ id: 'metrics-user' }, jwtSecret, {
        expiresIn: `${expiresIn}h`,
      });
      statsdCache.set('token', token);
    }

    // deep copy to ensure no changes to one object affects the other
    const toSend = _.cloneDeep(metricsBatch);
    // clear the batch _before_ the send so new metrics can be added
    metricsBatch = [];

    // actually write stats
    await thisInstance
      .post(`/batch`, toSend, {
        headers: { 'X-JWT-Token': token },
      })
      .then(() => {
        console.log(`Successfully wrote ${toSend.length} metrics to /batch`);
      })
      .catch((err) => {
        console.log(`Error sending ${toSend.length} metrics to /batch: ${err}`);
      });
  }
};

/*
 * Make sure we push metrics every X seconds
 * regardless of size of current batch
 *
 * separate from similar if statement to ensure
 * all variables are available to all functions
 */
if (endpoint && enabled) {
  console.log(`Scheduling metrics send for every ${batchTiming} ms`);
  setInterval(function () {
    sendMetrics(true).catch((e) =>
      console.log(`Failed to write timed stat batch: ${e}`)
    );
  }, batchTiming);
}

const processMetric = async (
  metric_type: string,
  metric: string,
  tags: { key: string; value: string }[],
  value: number,
  sampleRate?: number
): Promise<void> => {
  if (!endpoint || !enabled) return;
  /*
   * Add default tags to any tags the user provides
   * if the user hasn't provided any tags, we should
   * still apply the default tags.
   * Also make sure we don't add duplicate tags.
   */
  let allTags = `${defaultTags}`;
  if (tags) {
    tags.forEach((tag) => {
      const newTag = `${tag.key}=${tag.value}`;
      if (!allTags.includes(newTag)) {
        allTags += `,${newTag}`;
      }
    });
  }

  /*
   * The data object we'll send to the statsd proxy
   * If `sampleRate` doesn't exist, typescript removes the key
   */
  const data = { value, metric, metric_type, tags: allTags, sampleRate };
  metricsBatch.push(data);
  await sendMetrics(false);
};

/*
 * The actual functions people should call.
 * These functions clearly mark the type of metric
 * that the user is producing
 */
const sendCounter = async (
  metric: string,
  tags: { key: string; value: string }[],
  value: number,
  sampleRate?: number
): Promise<void> => {
  if (typeof sampleRate !== 'undefined') {
    await processMetric('count', metric, tags, value, sampleRate);
  } else {
    await processMetric('count', metric, tags, value);
  }
};

const sendGauge = async (
  metric: string,
  tags: { key: string; value: string }[],
  value: number
): Promise<void> => {
  await processMetric('gauge', metric, tags, value);
};

const sendTiming = async (
  metric: string,
  tags: { key: string; value: string }[],
  value: number,
  sampleRate?: number
): Promise<void> => {
  if (typeof sampleRate !== 'undefined') {
    await processMetric('timing', metric, tags, value, sampleRate);
  } else {
    await processMetric('timing', metric, tags, value);
  }
};

const sendSet = async (
  metric: string,
  tags: { key: string; value: string }[],
  value: number
): Promise<void> => {
  await processMetric('set', metric, tags, value);
};

export { sendCounter, sendGauge, sendTiming, sendSet };
