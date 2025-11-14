package main

import (
	"github.com/diwise/service-chassis/pkg/infrastructure/servicerunner"
)

type flagType int
type flagMap map[flagType]string

const (
	listenAddress flagType = iota
	servicePort
	controlPort

	deviceManagementUrl
	measurementsUrl
	functionsFilePath

	dbHost
	dbUser
	dbPassword
	dbPort
	dbName
	dbSSLMode

	oauth2ClientId
	oauth2ClientSecret
	oauth2TokenUrl
	oauth2InsecureURL
)

type appConfig struct{}

var oninit = servicerunner.OnInit[appConfig]
var onstarting = servicerunner.OnStarting[appConfig]
var onshutdown = servicerunner.OnShutdown[appConfig]
var webserver = servicerunner.WithHTTPServeMux[appConfig]
var muxinit = servicerunner.OnMuxInit[appConfig]
var listen = servicerunner.WithListenAddr[appConfig]
var port = servicerunner.WithPort[appConfig]
var pprof = servicerunner.WithPPROF[appConfig]
var liveness = servicerunner.WithK8SLivenessProbe[appConfig]
var readiness = servicerunner.WithK8SReadinessProbes[appConfig]
var withtracing = servicerunner.WithTracing[appConfig]
