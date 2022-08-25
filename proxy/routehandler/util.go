package routehandler

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	vmmetrics "github.com/VictoriaMetrics/metrics"
)

var (
	allowedNames     = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_:]*$")
	allowedFirstChar = regexp.MustCompile("^[a-zA-Z]")
	replaceChars     = regexp.MustCompile("[^a-zA-Z0-9_:]")
	allowedTagKeys   = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_]*$")
)

// 5 MB
const maxBodySize = 5000 * 1024 * 1024

func procBody(r *http.Request) ([]byte, error) {
	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		return []byte{}, fmt.Errorf("Unsupported content type %v", r.Header.Get("Content-Type"))
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		return []byte{}, err
	}
	r.Body.Close()

	return body, nil
}

func processMetric(m MetricRequest, prefix string, normalize bool, promFilter bool) (MetricRequest, error) {
	var err error
	if prefix != "" {
		m.Metric = prefix + m.Metric
	}

	if normalize {
		m.Metric = strings.ToLower(m.Metric)
		m.Tags = strings.ToLower(m.Tags)
	}

	if promFilter {
		m, err = filterPromMetric(m)
		if err != nil {
			return MetricRequest{}, err
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

func filterPromMetric(m MetricRequest) (MetricRequest, error) {
	/*
	Remove/Replace any characters that don't meet the Prometheus
	data model requirements: https://prometheus.io/docs/concepts/data_model/
	*/
	metric := MetricRequest{
		Metric: "", Value: m.Value, Tags: "", SampleRate: 0,
	}
	if !allowedFirstChar.MatchString(m.Metric) {
		vmmetrics.GetOrCreateCounter("metrics_dropped_total").Inc()
		return MetricRequest{}, fmt.Errorf("Invalid first character in metric name")
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
	metric.SampleRate = m.SampleRate
	log.WithFields(log.Fields{"metric": metric}).Debug("Final Metric")
	return metric, nil
}
