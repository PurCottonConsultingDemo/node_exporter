package process_detail

import (
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/node_exporter/collector/process_detail/common"
	"github.com/prometheus/node_exporter/collector/process_detail/proc"
	"strconv"
	"strings"
)

var (
	pidDesc = prometheus.NewDesc(
		"namedprocess_namegroup_pid",
		"process pid",
		[]string{"groupname", "processname"},
		nil)

	effectiveUsernameDesc = prometheus.NewDesc(
		"namedprocess_namegroup_effective_username",
		"effective username",
		[]string{"groupname", "username"},
		nil)

	netstatDesc = prometheus.NewDesc(
		"namedprocess_namegroup_netstat",
		"proc related netstat info",
		[]string{"groupname", "processname", "pid", "cmdline", "netstats"},
		nil)

	numprocsDesc = prometheus.NewDesc(
		"namedprocess_namegroup_num_procs",
		"number of processes in this group",
		[]string{"groupname"},
		nil)

	cpuSecsDesc = prometheus.NewDesc(
		"namedprocess_namegroup_cpu_seconds_total",
		"Cpu user usage in seconds",
		[]string{"groupname", "mode"},
		nil)

	readBytesDesc = prometheus.NewDesc(
		"namedprocess_namegroup_read_bytes_total",
		"number of bytes read by this group",
		[]string{"groupname"},
		nil)

	writeBytesDesc = prometheus.NewDesc(
		"namedprocess_namegroup_write_bytes_total",
		"number of bytes written by this group",
		[]string{"groupname"},
		nil)

	majorPageFaultsDesc = prometheus.NewDesc(
		"namedprocess_namegroup_major_page_faults_total",
		"Major page faults",
		[]string{"groupname"},
		nil)

	minorPageFaultsDesc = prometheus.NewDesc(
		"namedprocess_namegroup_minor_page_faults_total",
		"Minor page faults",
		[]string{"groupname"},
		nil)

	contextSwitchesDesc = prometheus.NewDesc(
		"namedprocess_namegroup_context_switches_total",
		"Context switches",
		[]string{"groupname", "ctxswitchtype"},
		nil)

	membytesDesc = prometheus.NewDesc(
		"namedprocess_namegroup_memory_bytes",
		"number of bytes of memory in use",
		[]string{"groupname", "memtype"},
		nil)

	openFDsDesc = prometheus.NewDesc(
		"namedprocess_namegroup_open_filedesc",
		"number of open file descriptors for this group",
		[]string{"groupname"},
		nil)

	worstFDRatioDesc = prometheus.NewDesc(
		"namedprocess_namegroup_worst_fd_ratio",
		"the worst (closest to 1) ratio between open fds and max fds among all procs in this group",
		[]string{"groupname"},
		nil)

	startTimeDesc = prometheus.NewDesc(
		"namedprocess_namegroup_oldest_start_time_seconds",
		"start time in seconds since 1970/01/01 of oldest process in group",
		[]string{"groupname"},
		nil)

	numThreadsDesc = prometheus.NewDesc(
		"namedprocess_namegroup_num_threads",
		"Number of threads",
		[]string{"groupname"},
		nil)

	statesDesc = prometheus.NewDesc(
		"namedprocess_namegroup_states",
		"Number of processes in states Running, Sleeping, Waiting, Zombie, or Other",
		[]string{"groupname", "state"},
		nil)

	scrapeErrorsDesc = prometheus.NewDesc(
		"namedprocess_scrape_errors",
		"general scrape errors: no proc metrics collected during a cycle",
		nil,
		nil)

	scrapeProcReadErrorsDesc = prometheus.NewDesc(
		"namedprocess_scrape_procread_errors",
		"incremented each time a proc's metrics collection fails",
		nil,
		nil)

	scrapePartialErrorsDesc = prometheus.NewDesc(
		"namedprocess_scrape_partial_errors",
		"incremented each time a tracked proc's metrics collection fails partially, e.g. unreadable I/O stats",
		nil,
		nil)

	threadWchanDesc = prometheus.NewDesc(
		"namedprocess_namegroup_threads_wchan",
		"Number of threads in this group waiting on each wchan",
		[]string{"groupname", "wchan"},
		nil)

	threadCountDesc = prometheus.NewDesc(
		"namedprocess_namegroup_thread_count",
		"Number of threads in this group with same threadname",
		[]string{"groupname", "threadname"},
		nil)

	threadCpuSecsDesc = prometheus.NewDesc(
		"namedprocess_namegroup_thread_cpu_seconds_total",
		"Cpu user/system usage in seconds",
		[]string{"groupname", "threadname", "mode"},
		nil)

	threadIoBytesDesc = prometheus.NewDesc(
		"namedprocess_namegroup_thread_io_bytes_total",
		"number of bytes read/written by these threads",
		[]string{"groupname", "threadname", "iomode"},
		nil)

	threadMajorPageFaultsDesc = prometheus.NewDesc(
		"namedprocess_namegroup_thread_major_page_faults_total",
		"Major page faults for these threads",
		[]string{"groupname", "threadname"},
		nil)

	threadMinorPageFaultsDesc = prometheus.NewDesc(
		"namedprocess_namegroup_thread_minor_page_faults_total",
		"Minor page faults for these threads",
		[]string{"groupname", "threadname"},
		nil)

	threadContextSwitchesDesc = prometheus.NewDesc(
		"namedprocess_namegroup_thread_context_switches_total",
		"Context switches for these threads",
		[]string{"groupname", "threadname", "ctxswitchtype"},
		nil)
)

type (
	scrapeRequest struct {
		results chan<- prometheus.Metric
		done    chan struct{}
	}

	ProcessCollectorOption struct {
		ProcFSPath  string
		Children    bool
		Threads     bool
		GatherSMaps bool
		Namer       common.MatchNamer
		Recheck     bool
		Debug       bool
	}

	NamedProcessCollector struct {
		scrapeChan           chan scrapeRequest
		gp                   *proc.Grouper
		threads              bool
		smaps                bool
		source               proc.Source
		scrapeErrors         int
		scrapeProcReadErrors int
		scrapePartialErrors  int
		debug                bool
		logger               log.Logger
	}
)

func NewProcessCollector(logger log.Logger, options ProcessCollectorOption) (*NamedProcessCollector, error) {
	fs, err := proc.NewFS(options.ProcFSPath, options.Debug)
	if err != nil {
		return nil, err
	}

	fs.GatherSMaps = options.GatherSMaps
	p := &NamedProcessCollector{
		scrapeChan: make(chan scrapeRequest),
		gp:         proc.NewGrouper(options.Namer, options.Children, options.Threads, options.Recheck, options.Debug, fs),
		source:     fs,
		threads:    options.Threads,
		smaps:      options.GatherSMaps,
		debug:      options.Debug,
	}

	colErrs, _, err := p.gp.Update(p.source.AllProcs())
	if err != nil {
		if options.Debug {
			_ = level.Error(logger).Log(err)
		}
		return nil, err
	}
	p.scrapePartialErrors += colErrs.Partial
	p.scrapeProcReadErrors += colErrs.Read

	return p, nil
}

// Update implements collector.Collector.
func (p *NamedProcessCollector) Update(ch chan<- prometheus.Metric) error {
	p.scrape(ch)

	return nil
}

func (p *NamedProcessCollector) scrape(ch chan<- prometheus.Metric) {
	permErrs, groups, err := p.gp.Update(p.source.AllProcs())
	p.scrapePartialErrors += permErrs.Partial
	if err != nil {
		p.scrapeErrors++
		_ = level.Error(p.logger).Log("error reading procs: %v", err)
	} else {
		for gname, gcounts := range groups {
			ch <- prometheus.MustNewConstMetric(pidDesc,
				prometheus.GaugeValue, float64(gcounts.Pid), gname, gcounts.ProcessName)
			ch <- prometheus.MustNewConstMetric(effectiveUsernameDesc,
				prometheus.GaugeValue, float64(0), gname, gcounts.EffectiveUsername)
			ch <- prometheus.MustNewConstMetric(numprocsDesc,
				prometheus.GaugeValue, float64(gcounts.Procs), gname)
			ch <- prometheus.MustNewConstMetric(membytesDesc,
				prometheus.GaugeValue, float64(gcounts.Memory.ResidentBytes), gname, "resident")
			ch <- prometheus.MustNewConstMetric(membytesDesc,
				prometheus.GaugeValue, float64(gcounts.Memory.VirtualBytes), gname, "virtual")
			ch <- prometheus.MustNewConstMetric(membytesDesc,
				prometheus.GaugeValue, float64(gcounts.Memory.VmSwapBytes), gname, "swapped")
			ch <- prometheus.MustNewConstMetric(startTimeDesc,
				prometheus.GaugeValue, float64(gcounts.OldestStartTime.Unix()), gname)
			ch <- prometheus.MustNewConstMetric(openFDsDesc,
				prometheus.GaugeValue, float64(gcounts.OpenFDs), gname)
			ch <- prometheus.MustNewConstMetric(worstFDRatioDesc,
				prometheus.GaugeValue, float64(gcounts.WorstFDratio), gname)
			ch <- prometheus.MustNewConstMetric(cpuSecsDesc,
				prometheus.CounterValue, gcounts.CPUUserTime, gname, "user")
			ch <- prometheus.MustNewConstMetric(cpuSecsDesc,
				prometheus.CounterValue, gcounts.CPUSystemTime, gname, "system")
			ch <- prometheus.MustNewConstMetric(readBytesDesc,
				prometheus.CounterValue, float64(gcounts.ReadBytes), gname)
			ch <- prometheus.MustNewConstMetric(writeBytesDesc,
				prometheus.CounterValue, float64(gcounts.WriteBytes), gname)
			ch <- prometheus.MustNewConstMetric(majorPageFaultsDesc,
				prometheus.CounterValue, float64(gcounts.MajorPageFaults), gname)
			ch <- prometheus.MustNewConstMetric(minorPageFaultsDesc,
				prometheus.CounterValue, float64(gcounts.MinorPageFaults), gname)
			ch <- prometheus.MustNewConstMetric(contextSwitchesDesc,
				prometheus.CounterValue, float64(gcounts.CtxSwitchVoluntary), gname, "voluntary")
			ch <- prometheus.MustNewConstMetric(contextSwitchesDesc,
				prometheus.CounterValue, float64(gcounts.CtxSwitchNonvoluntary), gname, "nonvoluntary")
			ch <- prometheus.MustNewConstMetric(numThreadsDesc,
				prometheus.GaugeValue, float64(gcounts.NumThreads), gname)
			ch <- prometheus.MustNewConstMetric(statesDesc,
				prometheus.GaugeValue, float64(gcounts.States.Running), gname, "Running")
			ch <- prometheus.MustNewConstMetric(statesDesc,
				prometheus.GaugeValue, float64(gcounts.States.Sleeping), gname, "Sleeping")
			ch <- prometheus.MustNewConstMetric(statesDesc,
				prometheus.GaugeValue, float64(gcounts.States.Waiting), gname, "Waiting")
			ch <- prometheus.MustNewConstMetric(statesDesc,
				prometheus.GaugeValue, float64(gcounts.States.Zombie), gname, "Zombie")
			ch <- prometheus.MustNewConstMetric(statesDesc,
				prometheus.GaugeValue, float64(gcounts.States.Other), gname, "Other")

			for wchan, count := range gcounts.Wchans {
				ch <- prometheus.MustNewConstMetric(threadWchanDesc,
					prometheus.GaugeValue, float64(count), gname, wchan)
			}

			if p.smaps {
				ch <- prometheus.MustNewConstMetric(membytesDesc,
					prometheus.GaugeValue, float64(gcounts.Memory.ProportionalBytes), gname, "proportionalResident")
				ch <- prometheus.MustNewConstMetric(membytesDesc,
					prometheus.GaugeValue, float64(gcounts.Memory.ProportionalSwapBytes), gname, "proportionalSwapped")
			}

			if p.threads {
				for _, thr := range gcounts.Threads {
					ch <- prometheus.MustNewConstMetric(threadCountDesc,
						prometheus.GaugeValue, float64(thr.NumThreads),
						gname, thr.Name)
					ch <- prometheus.MustNewConstMetric(threadCpuSecsDesc,
						prometheus.CounterValue, float64(thr.CPUUserTime),
						gname, thr.Name, "user")
					ch <- prometheus.MustNewConstMetric(threadCpuSecsDesc,
						prometheus.CounterValue, float64(thr.CPUSystemTime),
						gname, thr.Name, "system")
					ch <- prometheus.MustNewConstMetric(threadIoBytesDesc,
						prometheus.CounterValue, float64(thr.ReadBytes),
						gname, thr.Name, "read")
					ch <- prometheus.MustNewConstMetric(threadIoBytesDesc,
						prometheus.CounterValue, float64(thr.WriteBytes),
						gname, thr.Name, "write")
					ch <- prometheus.MustNewConstMetric(threadMajorPageFaultsDesc,
						prometheus.CounterValue, float64(thr.MajorPageFaults),
						gname, thr.Name)
					ch <- prometheus.MustNewConstMetric(threadMinorPageFaultsDesc,
						prometheus.CounterValue, float64(thr.MinorPageFaults),
						gname, thr.Name)
					ch <- prometheus.MustNewConstMetric(threadContextSwitchesDesc,
						prometheus.CounterValue, float64(thr.CtxSwitchVoluntary),
						gname, thr.Name, "voluntary")
					ch <- prometheus.MustNewConstMetric(threadContextSwitchesDesc,
						prometheus.CounterValue, float64(thr.CtxSwitchNonvoluntary),
						gname, thr.Name, "nonvoluntary")
				}
			}

			var netstats []string
			for _, netInfo := range gcounts.Nets {
				bld := strings.Builder{}
				bld.WriteString(netInfo.Proto)
				bld.WriteString("/")
				bld.WriteString(netInfo.LocalIpPort)
				bld.WriteString("/")
				bld.WriteString(netInfo.RemoteIpPort)
				bld.WriteString("/")
				bld.WriteString(strconv.FormatUint(netInfo.State, 16))

				netstats = append(netstats, bld.String())
			}

			ch <- prometheus.MustNewConstMetric(netstatDesc,
				prometheus.GaugeValue, 0,
				gname,
				gcounts.ProcessName,
				strconv.Itoa(gcounts.Pid),
				strings.Join(gcounts.CmdLine, " "),
				strings.Join(netstats, ";"))
		}
	}
	ch <- prometheus.MustNewConstMetric(scrapeErrorsDesc,
		prometheus.CounterValue, float64(p.scrapeErrors))
	ch <- prometheus.MustNewConstMetric(scrapeProcReadErrorsDesc,
		prometheus.CounterValue, float64(p.scrapeProcReadErrors))
	ch <- prometheus.MustNewConstMetric(scrapePartialErrorsDesc,
		prometheus.CounterValue, float64(p.scrapePartialErrors))
}
