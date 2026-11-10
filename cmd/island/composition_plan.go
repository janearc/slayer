package main

// These are authored scene controls. Source heights and geographic references
// retain their separate contracts; a plant mixture is not a vegetation survey.
type geographicPoint struct {
	Longitude float64 `yaml:"longitude-deg" json:"longitude-deg"`
	Latitude  float64 `yaml:"latitude-deg" json:"latitude-deg"`
}

type localBounds struct {
	MinEast  float64 `yaml:"min-east" json:"min-east"`
	MinSouth float64 `yaml:"min-south" json:"min-south"`
	MaxEast  float64 `yaml:"max-east" json:"max-east"`
	MaxSouth float64 `yaml:"max-south" json:"max-south"`
}

type heightBand struct {
	Below  float64    `yaml:"below-metres" json:"below-metres"`
	Colour [3]float64 `yaml:"colour-rgb" json:"colour-rgb"`
}

type objectPlan struct {
	Name    string  `yaml:"name" json:"name"`
	Kind    string  `yaml:"kind" json:"kind"`
	Seed    uint64  `yaml:"seed" json:"seed"`
	Detail  string  `yaml:"detail" json:"detail"`
	East    float64 `yaml:"east-metres" json:"east-metres"`
	South   float64 `yaml:"south-metres" json:"south-metres"`
	Above   float64 `yaml:"above-metres" json:"above-metres"`
	Heading float64 `yaml:"heading-deg" json:"heading-deg"`
}

type forestPlan struct {
	AreaSide        int         `yaml:"aggregate-area-samples-side" json:"aggregate-area-samples-side,omitempty"`
	GeometryMinimum string      `yaml:"minimum-geometry-detail" json:"minimum-geometry-detail,omitempty"`
	Adaptive        bool        `yaml:"adaptive-regions" json:"adaptive-regions"`
	EdgeFade        float64     `yaml:"edge-fade-metres" json:"edge-fade-metres"`
	HeightFade      float64     `yaml:"height-fade-metres" json:"height-fade-metres"`
	AggregateColour [3]float64  `yaml:"aggregate-colour-rgb" json:"aggregate-colour-rgb"`
	Name            string      `yaml:"name" json:"name"`
	Seed            uint64      `yaml:"seed" json:"seed"`
	Bounds          localBounds `yaml:"bounds-metres" json:"bounds-metres"`
	Cell            float64     `yaml:"cell-metres" json:"cell-metres"`
	Density         float64     `yaml:"density-per-hectare" json:"density-per-hectare"`
	Variety         int         `yaml:"variety" json:"variety"`
	Mixture         []struct {
		Kind   string  `yaml:"kind" json:"kind"`
		Weight float64 `yaml:"weight" json:"weight"`
	} `yaml:"mixture" json:"mixture"`
	MinimumHeight float64    `yaml:"minimum-height-metres" json:"minimum-height-metres"`
	MaximumHeight float64    `yaml:"maximum-height-metres" json:"maximum-height-metres"`
	Clearance     float64    `yaml:"clearance-metres" json:"clearance-metres"`
	Radius        float64    `yaml:"maximum-radius-metres" json:"maximum-radius-metres"`
	Height        float64    `yaml:"maximum-tree-height-metres" json:"maximum-tree-height-metres"`
	Thresholds    [4]float64 `yaml:"detail-thresholds-pixels" json:"detail-thresholds-pixels"`
}

type mistPlan struct {
	EdgeFade        float64     `yaml:"edge-fade-metres" json:"edge-fade-metres"`
	Bounds          localBounds `yaml:"bounds-metres" json:"bounds-metres"`
	Base            float64     `yaml:"base-metres" json:"base-metres"`
	Depth           float64     `yaml:"depth-metres" json:"depth-metres"`
	Extinction      float64     `yaml:"extinction-per-metre" json:"extinction-per-metre"`
	Coverage        float64     `yaml:"coverage" json:"coverage"`
	Seed            uint64      `yaml:"seed" json:"seed"`
	Wind            [3]float64  `yaml:"wind-metres-per-second" json:"wind-metres-per-second"`
	Colour          [3]float64  `yaml:"colour-rgb" json:"colour-rgb"`
	FloorSide       int         `yaml:"floor-side" json:"floor-side"`
	FloorTolerance  float64     `yaml:"floor-tolerance-metres" json:"floor-tolerance-metres"`
	FloorIterations int         `yaml:"floor-iterations" json:"floor-iterations"`
}

type laserPlan struct {
	Origins    [][3]float64 `yaml:"origins-metres" json:"origins-metres"`
	Azimuth    float64      `yaml:"azimuth-deg" json:"azimuth-deg"`
	Altitude   float64      `yaml:"altitude-deg" json:"altitude-deg"`
	Radius     float64      `yaml:"radius-metres" json:"radius-metres"`
	Reach      float64      `yaml:"reach-metres" json:"reach-metres"`
	Divergence float64      `yaml:"divergence-deg" json:"divergence-deg"`
	Wavelength float64      `yaml:"wavelength-nm" json:"wavelength-nm"`
	Strength   float64      `yaml:"display-strength" json:"display-strength"`
	Launchers  bool         `yaml:"visible-launchers" json:"visible-launchers"`
}

type sitePlan struct {
	LightExposure float64         `yaml:"light-exposure" json:"light-exposure"`
	At            geographicPoint `yaml:"at" json:"at"`
	Objects       []objectPlan    `yaml:"objects" json:"objects"`
	Forest        *forestPlan     `yaml:"forest" json:"forest,omitempty"`
	Atmosphere    []mistPlan      `yaml:"atmosphere" json:"atmosphere,omitempty"`
	Lasers        *laserPlan      `yaml:"lasers" json:"lasers,omitempty"`
	Budget        struct {
		Cells      int `yaml:"cells" json:"cells"`
		Candidates int `yaml:"candidates" json:"candidates"`
		Instances  int `yaml:"instances" json:"instances"`
		Faces      int `yaml:"prototype-faces" json:"prototype-faces"`
	} `yaml:"budget" json:"budget"`
}
