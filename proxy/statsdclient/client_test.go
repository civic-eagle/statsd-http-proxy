
package statsdclient

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewGoMetricClient(t *testing.T) {
	client := NewGoMetricClient("127.0.0.1", 8125)

	require.Equal(t, "*statsd.Client", reflect.TypeOf(client).String())
}
