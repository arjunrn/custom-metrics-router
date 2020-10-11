package routes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMetricServiceList(t *testing.T) {
	for _, tc := range []struct {
		name              string
		inputAPIServices  []MetricsAPIService
		outputAPIServices []MetricsAPIService
		deleteAPIServices []MetricsAPIService
	}{
		{
			name: "basic",
			inputAPIServices: []MetricsAPIService{
				{Name: "test", Namespace: "testns", Created: time.Unix(1, 0), Priority: 1},
			},
			outputAPIServices: []MetricsAPIService{
				{Name: "test", Namespace: "testns", Created: time.Unix(1, 0), Priority: 1},
			},
		},
		{
			name: "time stamp",
			inputAPIServices: []MetricsAPIService{
				{Name: "test1", Namespace: "testns", Created: time.Unix(3, 0), Priority: 1},
				{Name: "test2", Namespace: "testns", Created: time.Unix(1, 0), Priority: 1},
				{Name: "test3", Namespace: "testns", Created: time.Unix(2, 0), Priority: 1},
			},
			outputAPIServices: []MetricsAPIService{
				{Name: "test2", Namespace: "testns", Created: time.Unix(1, 0), Priority: 1},
				{Name: "test3", Namespace: "testns", Created: time.Unix(2, 0), Priority: 1},
				{Name: "test1", Namespace: "testns", Created: time.Unix(3, 0), Priority: 1},
			},
		},
		{
			name: "priority",
			inputAPIServices: []MetricsAPIService{
				{Name: "test1", Namespace: "testns", Created: time.Unix(1, 0), Priority: 3},
				{Name: "test2", Namespace: "testns", Created: time.Unix(1, 0), Priority: 1},
				{Name: "test3", Namespace: "testns", Created: time.Unix(1, 0), Priority: 2},
			},
			outputAPIServices: []MetricsAPIService{
				{Name: "test2", Namespace: "testns", Created: time.Unix(1, 0), Priority: 1},
				{Name: "test3", Namespace: "testns", Created: time.Unix(1, 0), Priority: 2},
				{Name: "test1", Namespace: "testns", Created: time.Unix(1, 0), Priority: 3},
			},
		},
		{
			name: "deletion",
			inputAPIServices: []MetricsAPIService{
				{Name: "test1", Namespace: "testns", Created: time.Unix(1, 0), Priority: 1},
				{Name: "test2", Namespace: "testns", Created: time.Unix(2, 0), Priority: 1},
				{Name: "test3", Namespace: "testns", Created: time.Unix(3, 0), Priority: 1},
			},
			deleteAPIServices: []MetricsAPIService{
				{Name: "test2", Namespace: "testns"},
			},
			outputAPIServices: []MetricsAPIService{
				{Name: "test1", Namespace: "testns", Created: time.Unix(1, 0), Priority: 1},
				{Name: "test3", Namespace: "testns", Created: time.Unix(3, 0), Priority: 1},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			list := make(MetricServiceList, 0)
			for _, s := range tc.inputAPIServices {
				list.AddService(s.Name, s.Namespace, s.Created, s.Priority)
			}
			for _, s := range tc.deleteAPIServices {
				list.RemoveService(s.Namespace, s.Name)
			}
			require.Len(t, list, len(tc.outputAPIServices))
			for i, o := range list {
				require.EqualValues(t, tc.outputAPIServices[i], o)
			}
		})
	}
}
