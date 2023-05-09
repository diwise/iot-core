package lwm2m

const lwm2mPrefix string = "urn:oma:lwm2m:ext:"

const (
	DigitalInput string = lwm2mPrefix + "3200"
	Presence     string = lwm2mPrefix + "3302"
	Temperature  string = lwm2mPrefix + "3303"
	Pressure     string = lwm2mPrefix + "3323"
	Conductivity string = lwm2mPrefix + "3327"
	Distance     string = lwm2mPrefix + "3330"
	AirQuality   string = lwm2mPrefix + "3428"
	Watermeter   string = lwm2mPrefix + "3424"
	Power        string = lwm2mPrefix + "3328"
	Energy       string = lwm2mPrefix + "3331"
)
