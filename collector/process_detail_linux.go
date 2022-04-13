package collector

import (
	"github.com/go-kit/log"
	"github.com/prometheus/node_exporter/collector/process_detail"
	"github.com/prometheus/node_exporter/collector/process_detail/common"
	"strconv"
)

func init() {
	registerCollector("process_detail", defaultEnabled, NewProcessDetailCollector)
}

func NewProcessDetailCollector(logger log.Logger) (Collector, error) {
	return process_detail.NewProcessCollector(
		logger,
		process_detail.ProcessCollectorOption{
			ProcFSPath:  "/proc",
			Children:    true,
			Threads:     false,
			GatherSMaps: true,
			Namer:       &matchAllWithPid{},
			Recheck:     false,
			Debug:       false,
		},
	)
}

type matchAllWithPid struct{}

const hyphen = "_"

func (m *matchAllWithPid) MatchAndName(nacl common.ProcAttributes) (bool, string) {
	return true, nacl.Name + hyphen + strconv.Itoa(nacl.PID)
}

func (m *matchAllWithPid) String() string {
	return "match all"
}
