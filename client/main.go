package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/client/common"
)

var log = logging.MustGetLogger("log")

// InitConfig Function that uses viper library to parse configuration parameters.
// Viper is configured to read variables from both environment variables and the
// config file ./config.yaml. Environment variables takes precedence over parameters
// defined in the configuration file. If some of the variables cannot be parsed,
// an error is returned
func InitConfig() (*viper.Viper, error) {
	v := viper.New()

	// Configure viper to read env variables with the CLI_ prefix
	v.AutomaticEnv()
	v.SetEnvPrefix("cli")
	// Use a replacer to replace env variables underscores with points. This let us
	// use nested configurations in the config file and at the same time define
	// env variables for the nested configurations
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Add env variables supported

	if v.BindEnv("id") != nil {
		fmt.Printf("Error loading env var %v", "id")
	}
	if v.BindEnv("agency") != nil {
		fmt.Printf("Error loading env var %v", "id")
	}
	if v.BindEnv("server", "address") != nil {
		fmt.Printf("Error loading env var %v.%v", "server", "address")
	}
	if v.BindEnv("loop", "period") != nil {
		fmt.Printf("Error loading env var %v.%v", "loop", "period")
	}
	if v.BindEnv("loop", "amount") != nil {
		fmt.Printf("Error loading env var %v.%v", "loop", "amount")
	}
	if v.BindEnv("log", "level") != nil {
		fmt.Printf("Error loading env var %v.%v", "log", "level")
	}
	if v.BindEnv("bet", "name") != nil {
		fmt.Printf("Error loading env var %v.%v", "bet", "name")
	}
	if v.BindEnv("bet", "surname") != nil {
		fmt.Printf("Error loading env var %v.%v", "bet", "surname")
	}
	if v.BindEnv("bet", "dni") != nil {
		fmt.Printf("Error loading env var %v.%v", "bet", "dni")
	}
	if v.BindEnv("bet", "birthdate") != nil {
		fmt.Printf("Error loading env var %v.%v", "bet", "birthdate")
	}
	if v.BindEnv("bet", "number") != nil {
		fmt.Printf("Error loading env var %v.%v", "bet", "number")
	}

	// Try to read configuration from config file. If config file
	// does not exist then ReadInConfig will fail but configuration
	// can be loaded from the environment variables so we shouldn't
	// return an error in that case
	v.SetConfigFile("./config.yaml")
	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("Configuration could not be read from config file. Using env variables instead")
	}

	// Parse time.Duration variables and return an error if those variables cannot be parsed

	if _, err := time.ParseDuration(v.GetString("loop.period")); err != nil {
		return nil, errors.Wrapf(err, "Could not parse CLI_LOOP_PERIOD env var as time.Duration.")
	}

	return v, nil
}

// InitLogger Receives the log level to be set in go-logging as a string. This method
// parses the string and set the level to the logger. If the level string is not
// valid an error is returned
func InitLogger(logLevel string) error {
	baseBackend := logging.NewLogBackend(os.Stdout, "", 0)
	format := logging.MustStringFormatter(
		`%{time:2006-01-02 15:04:05} %{level:.5s}     %{message}`,
	)
	backendFormatter := logging.NewBackendFormatter(baseBackend, format)

	backendLeveled := logging.AddModuleLevel(backendFormatter)
	logLevelCode, err := logging.LogLevel(logLevel)
	if err != nil {
		return err
	}
	backendLeveled.SetLevel(logLevelCode, "")

	// Set the backends to be used.
	logging.SetBackend(backendLeveled)
	return nil
}

// PrintConfig Print all the configuration parameters of the program.
// For debugging purposes only
func PrintConfig(v *viper.Viper) {
	log.Infof("action: config | result: success | client_id: %s | server_address: %s | loop_amount: %v | loop_period: %v | log_level: %s",
		v.GetString("id"),
		v.GetString("server.address"),
		v.GetInt("loop.amount"),
		v.GetDuration("loop.period"),
		v.GetString("log.level"),
	)
}

func main() {
	v, err := InitConfig()
	if err != nil {
		log.Criticalf("%s", err)
	}

	if err := InitLogger(v.GetString("log.level")); err != nil {
		log.Criticalf("%s", err)
	}

	// Print program config with debugging purposes
	PrintConfig(v)

	clientConfig := common.ClientConfig{
		ServerAddress: v.GetString("server.address"),
		ID:            v.GetString("id"),
		LoopAmount:    v.GetInt("loop.amount"),
		LoopPeriod:    v.GetDuration("loop.period"),
	}

	bet := common.Bet{
		Name:      v.GetString("bet.name"),
		Surname:   v.GetString("bet.surname"),
		Dni:       v.GetString("bet.dni"),
		BirthDate: v.GetString("bet.birthdate"),
		Number:    v.GetString("bet.number"),
		Agency:    v.GetString("agency"),
	}

	client := common.NewClient(clientConfig, bet)
	client.StartClientLoop()
}
