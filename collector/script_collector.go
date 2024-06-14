package collector

import (
	"os"
	"strconv"
	"strings"
	"os/exec"
	"path/filepath"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	scriptsDirectory = kingpin.Flag("collector.script.directory", "Directory contains scripts to execute.").Default("").String()
)

type scriptCollector struct {
	scriptDir 	string
	logger      log.Logger
}

func init() {
	registerCollector("script", defaultEnabled, NewScriptCollector)
}

// NewScriptCollector returns a new Collector exposing metrics returned from scripts
// in the given directory
func NewScriptCollector(logger log.Logger) (Collector, error) {
	c := &scriptCollector{
		scriptDir: *scriptsDirectory,
		logger: 	logger,
	}
	return c, nil
}

// Update implements the Collector interface.
func (c *scriptCollector) Update(ch chan<- prometheus.Metric) error {

	paths, err := filepath.Glob(c.scriptDir)
	if err != nil || len(paths) == 0 {
		// not glob or not accessible path either way assume single
		// directory and let os.ReadDir handle it
		paths = []string{c.scriptDir}
	}

	for _, path := range paths {
		files, err := os.ReadDir(path)
		if err != nil && path != "" {
			level.Error(c.logger).Log("msg", "failed to read scripts collector directory", "path", path, "err", err)
		}

		for _, f := range files {
			scriptPath := filepath.Join(path, f.Name())
			if !strings.HasSuffix(f.Name(), ".sh") {
				continue
			}

			output, err := exec.Command(scriptPath).Output()
			if err != nil {
				level.Error(c.logger).Log("msg", "failed to exectue script", "path", scriptPath, "err", err)
				continue
			}

			metricValue, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
			if err != nil {
				level.Error(c.logger).Log("msg", "error converting script output", "path", scriptPath, "err", err)
				continue
			}
			metricName := f.Name()[:strings.IndexByte(f.Name(), '.')]
			desc := prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "script", metricName),
				"Custom metric collected from script",
				nil, nil,
			)
			metric, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, metricValue)
			if err != nil {
				level.Error(c.logger).Log("msg", "Error creating metric for script", "path", scriptPath, "err", err)
				continue
			}
			ch <- metric
		}
	}

	return nil
}
