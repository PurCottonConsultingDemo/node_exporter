package collector

import (
	"github.com/go-kit/log"
	"github.com/prometheus/node_exporter/collector/process_detail"
	"github.com/prometheus/node_exporter/collector/process_detail/common"
)

func init() {
	registerCollector("process_detail", defaultDisabled, NewProcessDetailCollector)
}

func NewProcessDetailCollector(logger log.Logger) (Collector, error) {
	return process_detail.NewProcessCollector(
		logger,
		process_detail.ProcessCollectorOption{
			ProcFSPath:  "/proc",
			Children:    true,
			Threads:     true,
			GatherSMaps: true,
			Namer:       &matchAll{},
			Recheck:     false,
			Debug:       false,
		},
	)
}

type matchAll struct{}

func (m *matchAll) MatchAndName(nacl common.ProcAttributes) (bool, string) {
	return true, nacl.Name
}

func (m *matchAll) String() string {
	return "match all"
}
