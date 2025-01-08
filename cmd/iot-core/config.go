package main

import (
	"github.com/diwise/iot-core/internal/pkg/application/functions"
	"github.com/diwise/iot-core/internal/pkg/application/measurements"
	"github.com/diwise/iot-core/internal/pkg/infrastructure/database"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/servicerunner"
)

type FlagType int
type FlagMap map[FlagType]string

const (
	listenAddress FlagType = iota
	servicePort
	controlPort
	configFilePath
	devMgmtUrl
	measurementsUrl
	policiesFile
	tokenUrl
	clientId
	clientSecret
	dbHost
	dbUser
	dbPassword
	dbPort
	dbName
	dbSslMode
	devMode
)

type AppConfig struct {
	messenger          messaging.MsgContext
	devMgmtClient      client.DeviceManagementClient
	measurementsClient measurements.MeasurementsClient
	storage            database.Storage
	registry           functions.Registry
}

var onstarting = servicerunner.OnStarting[AppConfig]
var onshutdown = servicerunner.OnShutdown[AppConfig]
var webserver = servicerunner.WithHTTPServeMux[AppConfig]
var muxinit = servicerunner.OnMuxInit[AppConfig]
var listen = servicerunner.WithListenAddr[AppConfig]
var port = servicerunner.WithPort[AppConfig]
var pprof = servicerunner.WithPPROF[AppConfig]
var liveness = servicerunner.WithK8SLivenessProbe[AppConfig]
var readiness = servicerunner.WithK8SReadinessProbes[AppConfig]
