package metadata

type Metadata struct {
	Name string `json:"name"`
	Unit string `json:"unit"`

	CosAlpha    *float64 `json:"cosAlpha,omitempty"`
	MaxDistance *float64 `json:"maxdistance,omitempty"`
	MaxLevel    *float64 `json:"maxlevel,omitempty"`
	MeanLevel   *float64 `json:"meanlevel,omitempty"`
}
