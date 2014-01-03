package opstocat

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
)

var environments = []string{"test", "development", "staging", "production", "enterprise"}

type Configuration struct {
	App              string
	Env              string `json:"APP_ENV"`
	AppConfigPath    string `json:"APP_CONFIG_PATH"`
	PidPath          string `json:"APP_PIDPATH"`
	LogFile          string `json:"APP_LOG_FILE"`
	StatsDAddress    string `json:"STATSD"`
	ForceStats       string `json:"FORCE_STATS"`
	HaystackUser     string `json:"FAILBOT_USERNAME"`
	HaystackPassword string `json:"FAILBOT_PASSWORD"`
	HaystackEndpoint string `json:"FAILBOT_URL"`
	SyslogAddr       string `json:"SYSLOG_ADDR"`
	Sha              string
	Hostname         string
}

type ConfigWrapper interface {
	OpstocatConfiguration() *Configuration
	SetupLogger()
}

func NewConfiguration(workingdir string) *Configuration {
	return &Configuration{
		App:           "",
		Env:           "",
		LogFile:       "",
		AppConfigPath: filepath.Join(workingdir, ".app-config"),
		Sha:           simpleExec("git", "rev-parse", "HEAD"),
		PidPath:       filepath.Join(workingdir, "tmp", "pids"),
		Hostname:      simpleExec("hostname", "-s"),
	}
}

func (c *Configuration) ShowPeriodicStats() bool {
	return len(c.StatsDAddress) > 0 || c.ForceStats == "1"
}

// Fill out the app config from values in the environment.  The env keys are
// the same as the json marshal keys in the config struct.
func ReadEnv(config ConfigWrapper) {
	innerconfig := config.OpstocatConfiguration()

	readEnvConfig(config)
	readEnvConfig(innerconfig)

	if innerconfig.Env == "" {
		ReadAppConfig(config, innerconfig.AppConfigPath)
	} else {
		ReadEnvConfig(config, innerconfig.AppConfigPath, innerconfig.Env)
	}
}

// Decode the environment's JSON file inside the appconfig path into the given
// config struct.
func ReadEnvConfig(config ConfigWrapper, appconfigPath, env string) error {
	return readAppConfig(config, filepath.Join(appconfigPath, env+".json"))
}

// Look for an available appconfig environment json file.
func ReadAppConfig(config ConfigWrapper, appconfigPath string) {
	var err error
	for _, env := range environments {
		err = ReadEnvConfig(config, appconfigPath, env)
		if err == nil {
			return
		}
	}
}

func readEnvConfig(config interface{}) {
	val := reflect.ValueOf(config).Elem()
	t := val.Type()
	fieldSize := val.NumField()
	for i := 0; i < fieldSize; i++ {
		valueField := val.Field(i)
		typeField := t.Field(i)
		envKey := typeField.Tag.Get("json")

		if envValue := os.Getenv(envKey); 0 < len(envValue) {
			valueField.SetString(envValue)
		}
	}
}

func readAppConfig(config ConfigWrapper, appconfig string) error {
	file, err := os.Open(appconfig)
	if err != nil {
		return err
	}

	defer file.Close()
	err = readAppConfigs(file, []interface{}{config, config.OpstocatConfiguration()})
	if err == nil {
		fmt.Printf("Reading app config %s\n", appconfig)
	} else {
		fmt.Printf("Error reading app config %s\n", appconfig)
		panic(err)
	}

	return err
}

func readAppConfigs(file *os.File, configs []interface{}) error {
	var err error
	dec := json.NewDecoder(file)
	for _, config := range configs {
		file.Seek(0, 0)
		err = dec.Decode(config)
		if err != nil {
			return err
		}
	}
	return nil
}

func simpleExec(name string, arg ...string) string {
	output, err := exec.Command(name, arg...).Output()
	if err != nil {
		panic(err)
	}

	return strings.Trim(string(output), " \n")
}
