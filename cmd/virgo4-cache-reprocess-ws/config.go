package main

import (
	"log"
	"os"
	"strconv"
)

// ServiceConfig defines all of the service configuration parameters
type ServiceConfig struct {
	ServicePort       int    // the service listen port
	OutQueueName      string // SQS queue name for outbound documents
	DataSourceName    string // the name to associate the data with. Each record has metadata showing this value
	MessageBucketName string // the bucket to use for large messages

	PostgresHost     string // the postgres endpoint
	PostgresPort     int    // and port
	PostgresUser     string // username
	PostgresPass     string // and password
	PostgresDatabase string // which database to use
	PostgresTable    string // which table to use

}

func ensureSet(env string) string {
	val, set := os.LookupEnv(env)

	if set == false {
		log.Printf("environment variable not set: [%s]", env)
		os.Exit(1)
	}

	return val
}

func ensureSetAndNonEmpty(env string) string {
	val := ensureSet(env)

	if val == "" {
		log.Printf("environment variable not set: [%s]", env)
		os.Exit(1)
	}

	return val
}

func envToInt(env string) int {

	number := ensureSetAndNonEmpty(env)
	n, err := strconv.Atoi(number)
	if err != nil {

		os.Exit(1)
	}
	return n
}

// LoadConfiguration will load the service configuration from env/cmdline
// and return a pointer to it. Any failures are fatal.
func LoadConfiguration() *ServiceConfig {

	var cfg ServiceConfig

	cfg.ServicePort = envToInt("VIRGO4_CACHE_REPROCESS_WS_PORT")
	cfg.OutQueueName = ensureSetAndNonEmpty("VIRGO4_CACHE_REPROCESS_WS_OUT_QUEUE")
	cfg.DataSourceName = ensureSetAndNonEmpty("VIRGO4_CACHE_REPROCESS_WS_DATA_SOURCE")
	cfg.MessageBucketName = ensureSetAndNonEmpty("VIRGO4_SQS_MESSAGE_BUCKET")

	cfg.PostgresHost = ensureSetAndNonEmpty("VIRGO4_CACHE_REPROCESS_WS_POSTGRES_HOST")
	cfg.PostgresPort = envToInt("VIRGO4_CACHE_REPROCESS_WS_POSTGRES_PORT")
	cfg.PostgresUser = ensureSetAndNonEmpty("VIRGO4_CACHE_REPROCESS_WS_POSTGRES_USER")
	cfg.PostgresPass = ensureSetAndNonEmpty("VIRGO4_CACHE_REPROCESS_WS_POSTGRES_PASS")
	cfg.PostgresDatabase = ensureSetAndNonEmpty("VIRGO4_CACHE_REPROCESS_WS_POSTGRES_DATABASE")
	cfg.PostgresTable = ensureSetAndNonEmpty("VIRGO4_CACHE_REPROCESS_WS_POSTGRES_TABLE")

	log.Printf("[CONFIG] ServicePort         = [%d]", cfg.ServicePort)
	log.Printf("[CONFIG] OutQueueName        = [%s]", cfg.OutQueueName)
	log.Printf("[CONFIG] DataSourceName      = [%s]", cfg.DataSourceName)
	log.Printf("[CONFIG] MessageBucketName   = [%s]", cfg.MessageBucketName)

	log.Printf("[CONFIG] PostgresHost        = [%s]", cfg.PostgresHost)
	log.Printf("[CONFIG] PostgresPort        = [%d]", cfg.PostgresPort)
	log.Printf("[CONFIG] PostgresUser        = [%s]", cfg.PostgresUser)
	log.Printf("[CONFIG] PostgresPass        = [REDACTED]")
	log.Printf("[CONFIG] PostgresDatabase    = [%s]", cfg.PostgresDatabase)
	log.Printf("[CONFIG] PostgresTable       = [%s]", cfg.PostgresTable)

	return &cfg
}

//
// end of file
//
