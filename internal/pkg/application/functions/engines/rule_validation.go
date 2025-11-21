package engines

type RuleValidation struct {
	MeasurementId      string
	DeviceId           string
	ShouldAbort        bool
	IsValid            bool
	ValidationMessages []string
}
