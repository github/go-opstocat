package opstocat

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/technoweenie/grohl"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type HaystackReporter struct {
	Endpoint string
	Hostname string
}

func NewHaystackReporter(config *Configuration) (*HaystackReporter, error) {
	endpoint, err := url.Parse(config.HaystackEndpoint)
	if err != nil {
		return nil, err
	}

	endpoint.User = url.UserPassword(config.HaystackUser, config.HaystackPassword)
	return &HaystackReporter{Endpoint: endpoint.String(), Hostname: config.Hostname}, nil
}

func (r *HaystackReporter) Report(err error, data grohl.Data) error {
	backtrace := grohl.ErrorBacktraceLines(err)
	data["backtrace"] = strings.Join(backtrace, "\n")
	data["host"] = r.Hostname
	data["rollup"] = r.rollup(data, backtrace[0])

	marshal, _ := json.Marshal(data)
	res, reporterr := http.Post(r.Endpoint, "application/json", bytes.NewBuffer(marshal))
	if res != nil {
		defer res.Body.Close()
	}

	if reporterr != nil || res.StatusCode != 201 {
		delete(data, "backtrace")
		delete(data, "host")
		if res != nil {
			data["haystackstatus"] = res.Status
		}
		grohl.Log(data)
		return reporterr
	}

	return nil
}

func (r *HaystackReporter) rollup(data grohl.Data, firstline string) string {
	hash := md5.New()
	io.WriteString(hash, fmt.Sprintf("%s:%s:%s", data["ns"], data["fn"], firstline))
	return fmt.Sprintf("%x", hash.Sum(nil))
}
