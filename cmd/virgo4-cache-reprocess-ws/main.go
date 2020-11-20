package main

import (
	"fmt"
	"log"
	"os"
	//"time"

	//"github.com/gin-contrib/cors"
	//"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

//
// main entry point
//
func main() {

	log.Printf("===> %s service staring up (version: %s) <===", os.Args[0], Version())

	// Get config params and use them to init service context. Any issues are fatal
	cfg := LoadConfiguration()

	svc := InitializeService(cfg)

	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()
	router := gin.Default()

	//corsCfg := cors.DefaultConfig()
	//corsCfg.AllowAllOrigins = true
	//corsCfg.AllowCredentials = true
	//corsCfg.AddAllowHeaders("Authorization")
	//router.Use(cors.New(corsCfg))

	p := ginprometheus.NewPrometheus("gin")

	// roundabout setup of /metrics endpoint to avoid double-gzip of response
	router.Use(p.HandlerFunc())
	h := promhttp.InstrumentMetricHandler(prometheus.DefaultRegisterer, promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{DisableCompression: true}))

	router.GET(p.MetricsPath, func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	})

	router.GET("/favicon.ico", svc.IgnoreHandler)

	router.GET("/version", svc.VersionHandler)
	router.GET("/healthcheck", svc.HealthCheckHandler)

	if api := router.Group("/api"); api != nil {
		api.PUT("/reindex/:id", svc.ReindexHandler)
	}

	portStr := fmt.Sprintf(":%d", cfg.ServicePort)
	log.Printf("Start service on %s", portStr)

	log.Fatal(router.Run(portStr))
}

//
// end of file
//
