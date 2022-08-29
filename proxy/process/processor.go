package processor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/civic-eagle/statsd-http-proxy/proxy/config"
	"github.com/civic-eagle/statsd-http-proxy/proxy/statsdclient"
	log "github.com/sirupsen/logrus"
	vmmetrics "github.com/VictoriaMetrics/metrics"
)

var (
	allowedNames     = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_:]*$")
	allowedFirstChar = regexp.MustCompile("^[a-zA-Z]")
	replaceChars     = regexp.MustCompile("[^a-zA-Z0-9_:]")
	allowedTagKeys   = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_]*$")

	counters = vmmetrics.NewCounter("counters_added_total")
	gauges = vmmetrics.NewCounter("gauges_added_total")
	timings = vmmetrics.NewCounter("timing_added_total")
	sets = vmmetrics.NewCounter("set_added_total")
)

// RouteHandler as a collection of route handlers
type Processor struct {
	statsdClient statsdclient.StatsdClientInterface
	metricPrefix string
	promFilter bool
	normalize bool
}

// NewProcessor creates tool to process metrics as they are submitted async
func NewProcessor(
	statsdClient statsdclient.StatsdClientInterface,
	metricPrefix string,
	promFilter bool,
	normalize bool,
) *Processor {
	// build processor
	processor := Processor{
		statsdClient,
		metricPrefix,
		promFilter,
		normalize,
	}

	return &processor
}

func (Processor *Processor) Process() {
	for msg := range config.ProcessChan {
		m, err := processMetric(msg, Processor.metricPrefix, Processor.normalize, Processor.promFilter)
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Failed to process metric")
			continue
		}
		Processor.sendMetric(m.MetricType, m.Metric, m.Value, float32(m.SampleRate))
	}
}

func (Processor *Processor) sendMetric(metricType string, key string, value int64, sampleRate float32) {
	/*
	Since we have two incoming handler paths for metrics
	we need a common switch case to actually process each metric
	once we've formatted it consistently
	Simply actually increment the correct values in our internal
	statsd client (and bump related internal metrics)
	*/
	switch metricType {
	case "count":
		Processor.statsdClient.Count(key, int(value), sampleRate)
		counters.Inc()
	case "gauge":
		Processor.statsdClient.Gauge(key, int(value))
		gauges.Inc()
	case "timing":
		Processor.statsdClient.Timing(key, value, sampleRate)
		timings.Inc()
	case "set":
		Processor.statsdClient.Set(key, int(value))
		sets.Inc()
	}
}

func processMetric(m config.MetricRequest, prefix string, normalize bool, promFilter bool) (config.MetricRequest, error) {
	var err error
	if prefix != "" {
		m.Metric = prefix + m.Metric
	}

	// An empty sample rate === a full sample rate
	if m.SampleRate == 0 {
		m.SampleRate = 1
	}

	if normalize {
		m.Metric = strings.ToLower(m.Metric)
		m.Tags = strings.ToLower(m.Tags)
	}

	if promFilter {
		m, err = filterPromMetric(m)
		if err != nil {
			return config.MetricRequest{}, err
		}
	}
	if m.Tags != "" {
		m.Metric += processTags(m.Tags)
	}

	return m, nil
}

func processTags(tagsList string) string {
	/*
	Process tags for a metric.
	Starting with a string of comma-separated key=value pairs,
	we need to ensure that:
	1. each pair contains two items, one on each side of a '='
	2. that both the key and the value strings have actual content ('=value' and 'key=' are not allowed)

	Then return the resulting pairs to a string

	Any failures means the metric gets no tags
	*/
	list := strings.Split(strings.TrimSpace(tagsList), ",")
	if len(list) == 0 {
		return ""
	}

	var finalTags string
	for _, pair := range list {
		pairItems := strings.Split(pair, "=")
		if len(pairItems) != 2 {
			vmmetrics.GetOrCreateCounter("metrics_tags_dropped_total").Inc()
			log.WithFields(log.Fields{"Tags": tagsList, "pair": pairItems}).Debug("Missing pair")
			continue
		} else if len(strings.TrimSpace(pairItems[0])) == 0 {
			vmmetrics.GetOrCreateCounter("metrics_tags_dropped_total").Inc()
			log.WithFields(log.Fields{"Tags": tagsList, "pair": pairItems}).Debug("Invalid tag key")
			continue
		} else if len(strings.TrimSpace(pairItems[1])) == 0 {
			vmmetrics.GetOrCreateCounter("metrics_tags_dropped_total").Inc()
			log.WithFields(log.Fields{"Tags": tagsList, "pair": pairItems}).Debug("Invalid tag value")
			continue
		}
		finalTags += fmt.Sprintf("%s,", pair)
	}
	return "," + strings.TrimSuffix(finalTags, ",")
}

func filterPromMetric(m config.MetricRequest) (config.MetricRequest, error) {
	/*
	Remove/Replace any characters that don't meet the Prometheus
	data model requirements: https://prometheus.io/docs/concepts/data_model/
	*/
	metric := config.MetricRequest{
		Metric: "", Value: m.Value, Tags: "", SampleRate: m.SampleRate,
		MetricType: m.MetricType,
	}
	if !allowedFirstChar.MatchString(m.Metric) {
		vmmetrics.GetOrCreateCounter("metrics_dropped_total").Inc()
		return metric, fmt.Errorf("Invalid first character in metric name")
	}
	if !allowedNames.MatchString(m.Metric) {
		metric.Metric = replaceChars.ReplaceAllString(m.Metric, "_")
	} else {
		metric.Metric = m.Metric
	}
	if m.Tags != "" {
		list := strings.Split(strings.TrimSpace(m.Tags), ",")
		if len(list) > 0 {
			var finalTags string
			for _, pair := range list {
				tagPair := strings.Split(pair, "=")
				// filter out any bad tag pairs first
				if len(tagPair) != 2 || len(strings.TrimSpace(tagPair[0])) == 0 || len(strings.TrimSpace(tagPair[1])) == 0 {
					log.WithFields(log.Fields{"Tags": list, "pair": tagPair}).Debug("Invalid tag set")
					continue
				}
				if !allowedTagKeys.MatchString(tagPair[0]) {
					tagKey := replaceChars.ReplaceAllString(tagPair[0], "_")
					finalTags += fmt.Sprintf("%s=%s,", tagKey, tagPair[1])
				} else {
					finalTags += fmt.Sprintf("%s,", pair)
				}
			}
			metric.Tags = strings.TrimSuffix(finalTags, ",")
		}
	} else {
		metric.Tags = ""
	}
	return metric, nil
}
