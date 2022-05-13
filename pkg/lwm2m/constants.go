package lwm2m

const lwm2mPrefix string = "urn:oma:lwm2m:ext:"

const (
	Presence    string = lwm2mPrefix + "3302"
	Temperature string = lwm2mPrefix + "3303"
	AirQuality  string = lwm2mPrefix + "3428"
)
