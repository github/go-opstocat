package opstocat

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func WritePid(config ConfigWrapper) {
	innerconfig := config.OpstocatConfiguration()
	pidfile := filepath.Join(innerconfig.PidPath, fmt.Sprintf("%s.pid", innerconfig.App))

	err := os.MkdirAll(innerconfig.PidPath, 0750)
	if err != nil {
		panic(err)
	}

	file, err := os.Create(pidfile)
	defer file.Close()

	if err != nil {
		panic(err)
	}

	pid := os.Getpid()
	file.WriteString(strconv.FormatInt(int64(pid), 10))
}
