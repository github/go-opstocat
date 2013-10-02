package opstocat

import (
	"fmt"
	"github.com/peterbourgon/g2s"
	"github.com/technoweenie/grohl"
	"log/syslog"
	"runtime"
	"strings"
	"time"
)

func SetupLogger(config ConfigWrapper) {
	innerconfig := config.OpstocatConfiguration()
	logch := make(chan grohl.Data, 100)
	chlogger, _ := grohl.NewChannelLogger(logch)
	grohl.SetLogger(chlogger)

	if len(innerconfig.StatsDAddress) > 0 {
		statter, _ := g2s.Dial("udp", innerconfig.StatsDAddress)
		grohl.CurrentStatter = statter
	}

	grohl.CurrentStatter = PrefixedStatter(innerconfig.App, grohl.CurrentStatter)

	if len(innerconfig.HaystackEndpoint) > 0 {
		grohl.CurrentContext.ExceptionReporter = NewHaystackReporter(innerconfig.HaystackEndpoint, innerconfig.Hostname)
	}

	grohl.AddContext("app", innerconfig.App)
	grohl.AddContext("deploy", innerconfig.Env)
	grohl.AddContext("sha", innerconfig.Sha)

	var logger grohl.Logger
	if len(innerconfig.SyslogAddr) > 0 {
		parts := strings.Split(innerconfig.SyslogAddr, ":")
		writer, err := syslog.Dial(parts[0], parts[1], syslog.LOG_INFO|syslog.LOG_LOCAL7, innerconfig.App)
		if err == nil {
			logger = grohl.NewIoLogger(writer)
		} else {
			grohl.Report(err, grohl.Data{"syslog_network": parts[0], "syslog_addr": parts[1]})
			fmt.Printf("Error opening syslog connection: %s\n", err)
		}
	}

	if logger == nil {
		logger = grohl.NewIoLogger(nil)
	}

	go grohl.Watch(logger, logch)
}

func SendPeriodicStats(duration string, config ConfigWrapper, callback func(keyprefix string)) error {
	innerconfig := config.OpstocatConfiguration()
	if !innerconfig.ShowPeriodicStats() {
		return nil
	}

	dur, err := time.ParseDuration(duration)
	if err != nil {
		return err
	}

	keyprefix := fmt.Sprintf("sys.%s.", innerconfig.Hostname)
	if callback == nil {
		callback = nopPeriodicCallback
	}

	go sendPeriodicStats(dur, keyprefix, callback)
	return nil
}

func sendPeriodicStats(dur time.Duration, keyprefix string, callback func(keyprefix string)) {
	for {
		time.Sleep(dur)
		grohl.Gauge(1.0, keyprefix+"goroutines", grohl.Format(runtime.NumGoroutine()))
		callback(keyprefix)
	}
}

func nopPeriodicCallback(keyprefix string) {}

func PrefixedStatter(prefix string, statter g2s.Statter) g2s.Statter {
	if prefix == "" {
		return statter
	}

	return &PrefixStatter{prefix, statter}
}

type PrefixStatter struct {
	Prefix  string
	Statter g2s.Statter
}

func (s *PrefixStatter) Counter(sampleRate float32, bucket string, n ...int) {
	s.Statter.Counter(sampleRate, fmt.Sprintf("%s.%s", s.Prefix, bucket), n...)
}

func (s *PrefixStatter) Timing(sampleRate float32, bucket string, d ...time.Duration) {
	s.Statter.Timing(sampleRate, fmt.Sprintf("%s.%s", s.Prefix, bucket), d...)
}

func (s *PrefixStatter) Gauge(sampleRate float32, bucket string, value ...string) {
	s.Statter.Gauge(sampleRate, fmt.Sprintf("%s.%s", s.Prefix, bucket), value...)
}
