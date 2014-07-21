package opstocat

import (
	"fmt"
	"github.com/peterbourgon/g2s"
	"github.com/technoweenie/grohl"
	"log/syslog"
	"net/url"
	"os"
	"runtime"
	"time"
)

func SetupLogger(config ConfigWrapper) {
	innerconfig := config.OpstocatConfiguration()
	logch := make(chan grohl.Data, 100)
	chlogger, _ := grohl.NewChannelLogger(logch)
	grohl.SetLogger(chlogger)

	if len(innerconfig.StatsDAddress) > 0 {
		if innerconfig.StatsDAddress == "noop" {
			grohl.CurrentStatter = &NoOpStatter{}
		} else {
			statter, err := g2s.Dial("udp", innerconfig.StatsDAddress)
			if err != nil {
				grohl.Report(err, grohl.Data{"statsd_address": innerconfig.StatsDAddress})
				grohl.CurrentStatter = &NoOpStatter{}
			} else {
				grohl.CurrentStatter = statter
			}
		}
	}

	grohl.CurrentStatter = PrefixedStatter(innerconfig.App, grohl.CurrentStatter)

	if len(innerconfig.HaystackEndpoint) > 0 {
		reporter, err := NewHaystackReporter(innerconfig)
		if err != nil {
			grohl.Report(err, grohl.Data{"haystack_enpdoint": innerconfig.HaystackEndpoint})
		} else {
			grohl.SetErrorReporter(reporter)
		}
	}

	grohl.AddContext("app", innerconfig.App)
	grohl.AddContext("deploy", innerconfig.Env)
	grohl.AddContext("sha", innerconfig.Sha)

	var logger grohl.Logger
	if len(innerconfig.SyslogAddr) > 0 {
		writer, err := newSyslogWriter(innerconfig.SyslogAddr, innerconfig.App)
		if err == nil {
			logger = grohl.NewIoLogger(writer)
		}
	}

	if logger == nil {
		logger = grohl.NewIoLogger(nil)
	}

	go grohl.Watch(logger, logch)
}

func newSyslogWriter(configAddr, tag string) (*syslog.Writer, error) {
	net, addr, err := parseAddr(configAddr)
	if err != nil {
		return nil, err
	}
	writer, err := syslog.Dial(net, addr, syslog.LOG_INFO|syslog.LOG_LOCAL7, tag)
	if err != nil {
		grohl.Report(err, grohl.Data{"syslog_network": net, "syslog_addr": addr})
		fmt.Fprintf(os.Stderr, "Error opening syslog connection: %s\n", err)
	}
	return writer, err
}

func parseAddr(s string) (string, string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", "", err
	}

	if u.Host == "" {
		return u.Scheme, u.Path, nil
	}
	return u.Scheme, u.Host, nil
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

type NoOpStatter struct{}

func (s *NoOpStatter) Counter(sampleRate float32, bucket string, n ...int)          {}
func (s *NoOpStatter) Timing(sampleRate float32, bucket string, d ...time.Duration) {}
func (s *NoOpStatter) Gauge(sampleRate float32, bucket string, value ...string)     {}
