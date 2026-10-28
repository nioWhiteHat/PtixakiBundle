package api

type PropertyFeatures struct {
	UID string

	SubType   string
	AdminName string

	SquareMeters      int
	AskingPricePerSqm *float64 `json:"AskingPricePerSqm,omitempty"`
	ConstructionYear  int
	RenovYear         int
	Distance          int
	IsRenovated       int

	ZoneValue         int
	AddressOSMType    string `json:"AddressOSMType,omitempty"`
	AddressOSMID      int64  `json:"AddressOSMID,omitempty"`
	Address           string
	GeoLat            float64
	GeoLon            float64
	NoView            int
	ViewForest        int
	ViewMountain      int
	ViewSea           int
	ViewUnlimited     int
	NoFacade          int
	FacadeCorner      int
	FacadeFrontage    int
	FacadeInterior    int
	FacadeThrough     int
	HeatType          string
	HeatFrom          string
	EnergyPerformance string
	WC                int
	AC                int
	Elevator          int
	Parking           int
	StorageRoom       int
	AluminumFrames    int
	Bedrooms          int
	Pool              int
	Garden            int
	FloorLevel        []int
}

type PropertyMlData struct {
	AgeSq                  int
	InteractionAgeDistance int
	LogInteraction         float64
}

type PropertyAdData struct {
	TransactionType string
	AskingPrice     int
	AdCreationAge   int
}

type PropertyPriceData struct {
	EstimatedPrice float64
	AskingPrice    float64
}
type PropertyImage struct {
	ImgUrl    string
	ImageData []byte
}
type PropertyData struct {
	Features  PropertyFeatures
	PriceData PropertyPriceData
	Images    []PropertyImage
}
