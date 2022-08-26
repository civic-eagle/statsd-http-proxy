package statsdclient

import GoMetricStatsdClient "github.com/GoMetric/go-statsd-client"

func NewGoMetricClient(
	statsdHost string,
	statsdPort int,
) StatsdClientInterface {
	return GoMetricStatsdClient.NewClient(statsdHost, statsdPort)
}

type StatsdClientInterface interface {
	Open()
	Close()
	Count(key string, value int, sampleRate float32)
	Timing(key string, time int64, sampleRate float32)
	Gauge(key string, value int)
	GaugeShift(key string, value int)
	Set(key string, value int)
}
