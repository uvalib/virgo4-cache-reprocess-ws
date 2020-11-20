package main

import (
	"github.com/uvalib/virgo4-sqs-sdk/awssqs"

	//"errors"
	//"fmt"
	"log"
	//"net"
	"net/http"
	//"path/filepath"
	//"runtime"
	//"strconv"
	//"strings"
	//"time"

	"github.com/gin-gonic/gin"
	//"github.com/uvalib/virgo4-jwt/v4jwt"
)

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
	//internalServiceError := false

	type hcResp struct {
		Healthy bool   `json:"healthy"`
		Message string `json:"message,omitempty"`
	}

	hcDB := hcResp{Healthy: true}
	//if ping != nil {
	//	internalServiceError = true
	//	hcSolr = hcResp{Healthy: false, Message: ping.Error()}
	//}

	hcMap := make(map[string]hcResp)
	hcMap["database"] = hcDB

	hcStatus := http.StatusOK
	//if internalServiceError == true {
	//	hcStatus = http.StatusInternalServerError
	//}

	c.JSON(hcStatus, hcMap)
}

func (svc *ServiceContext) ReindexHandler(c *gin.Context) {

	id := c.Param("id")

	_, err := svc.cache.Get(id)
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

	c.JSON(http.StatusOK, ServiceResponse{"OK"})
}

//
// end of file
//
