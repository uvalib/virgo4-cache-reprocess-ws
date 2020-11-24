package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/uvalib/virgo4-sqs-sdk/awssqs"
	//"github.com/uvalib/virgo4-jwt/v4jwt"
)

// number of times to retry a message put before giving up and terminating
var sendRetries = uint(3)

// this service may be used to reindex newly published items. In this case, the item identifier might not exist
// in a service cache so this attribute may be used to ignore the cache and lookup the item in an external
// service anyway. Typically done by the tracksys enrich service.
var ignoreCacheAttributeName = "ignore-cache"

// ServiceContext contains common data used by all handlers
type ServiceContext struct {
	config *ServiceConfig
	cache  CacheProxy
	aws    awssqs.AWS_SQS
	queue  awssqs.QueueHandle
}

type ServiceResponse struct {
	Message string `json:"msg"`
}

// InitializeService will initialize the service context based on the config parameters.
func InitializeService(cfg *ServiceConfig) *ServiceContext {
	log.Printf("initializing service")

	// load our AWS_SQS helper object
	aws, err := awssqs.NewAwsSqs(awssqs.AwsSqsConfig{MessageBucketName: cfg.MessageBucketName})
	fatalIfError(err)

	// our SQS output queue
	outQueueHandle, err := aws.QueueHandle(cfg.OutQueueName)
	fatalIfError(err)

	// our database cache proxy
	cacheProxy, err := NewCacheProxy(cfg)
	fatalIfError(err)

	svc := ServiceContext{
		config: cfg,
		cache:  cacheProxy,
		aws:    aws,
		queue:  outQueueHandle,
	}

	return &svc
}

// IgnoreHandler is a dummy to handle certain browser requests without warnings (e.g. favicons)
func (svc *ServiceContext) IgnoreHandler(c *gin.Context) {
}

// VersionHandler reports the version of the service
func (svc *ServiceContext) VersionHandler(c *gin.Context) {
	vMap := make(map[string]string)
	vMap["build"] = Version()
	c.JSON(http.StatusOK, vMap)
}

// HealthCheckHandler reports the health of the serivce
func (svc *ServiceContext) HealthCheckHandler(c *gin.Context) {

	type hcResp struct {
		Healthy bool   `json:"healthy"`
		Message string `json:"message,omitempty"`
	}

	dbHealthy := svc.cache.Healthcheck()
	hcDB := hcResp{Healthy: dbHealthy}
	hcMap := make(map[string]hcResp)
	hcMap["database"] = hcDB

	hcStatus := http.StatusOK
	if dbHealthy == false {
		hcStatus = http.StatusInternalServerError
	}

	c.JSON(hcStatus, hcMap)
}

func (svc *ServiceContext) ReindexHandler(c *gin.Context) {

	id := c.Param("id")

	// check the cache
	record, err := svc.cache.Get(id)
	if err != nil {
		if err == ErrNotInCache {
			log.Printf("WARNING: item not in cache: %s", id)
			c.JSON(http.StatusNotFound, ServiceResponse{"Not Found"})
		} else {
			log.Printf("ERROR: cache lookup: %s", err.Error())
			c.JSON(http.StatusInternalServerError, ServiceResponse{err.Error()})
		}
		return
	}

	// check the data source to ensure this is the correct item type for the configuration
	if record.Source != svc.config.DataSourceName {
		log.Printf("ERROR: incorrect source type for %s; expected %s, got %s", id, svc.config.DataSourceName, record.Source)
		c.JSON(http.StatusUnprocessableEntity, ServiceResponse{"Unprocessable Entity"})
		return
	}

	// send to outbound queue
	err = svc.queueOutbound(record)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ServiceResponse{err.Error()})
		return
	}

	log.Printf("INFO: item submitted for reindexing: %s", id)

	// all good
	c.JSON(http.StatusOK, ServiceResponse{"OK"})
}

func (svc *ServiceContext) queueOutbound(record *CacheRecord) error {
	outbound := svc.constructMessage(record.ID, record.Type, record.Source, record.Payload)
	messages := make([]awssqs.Message, 0, 1)
	messages = append(messages, *outbound)
	opStatus, err := svc.aws.BatchMessagePut(svc.queue, messages)
	if err != nil {
		// if an error we can handle, retry
		if err == awssqs.ErrOneOrMoreOperationsUnsuccessful {
			log.Printf("WARNING: item failed to send to the work queue, retrying...")

			// retry the failed item and bail out if we cannot retry
			err = svc.aws.MessagePutRetry(svc.queue, messages, opStatus, sendRetries)
		}
	}

	return err
}

// construct the outbound SQS message
func (svc *ServiceContext) constructMessage(id string, theType string, source string, payload string) *awssqs.Message {

	attributes := make([]awssqs.Attribute, 0, 5)
	attributes = append(attributes, awssqs.Attribute{Name: ignoreCacheAttributeName, Value: "true"})
	attributes = append(attributes, awssqs.Attribute{Name: awssqs.AttributeKeyRecordId, Value: id})
	attributes = append(attributes, awssqs.Attribute{Name: awssqs.AttributeKeyRecordType, Value: theType})
	attributes = append(attributes, awssqs.Attribute{Name: awssqs.AttributeKeyRecordSource, Value: source})
	attributes = append(attributes, awssqs.Attribute{Name: awssqs.AttributeKeyRecordOperation, Value: awssqs.AttributeValueRecordOperationUpdate})
	return &awssqs.Message{Attribs: attributes, Payload: []byte(payload)}
}

//
// end of file
//
