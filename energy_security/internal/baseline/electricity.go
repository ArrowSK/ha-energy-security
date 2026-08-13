package baseline

import "strings"

const electricityReferenceSource = "Ember country overview yearly dataset"

type ElectricityReference struct {
	Country            string  `json:"country"`
	Year               int     `json:"year"`
	PopulationMillions float64 `json:"population_millions"`
	DemandTWh          float64 `json:"demand_twh"`
	DemandMWhPerCapita float64 `json:"demand_mwh_per_capita"`
}

func (r ElectricityReference) AverageLoadMW() float64 {
	if r.DemandTWh <= 0 {
		return 0
	}
	return r.DemandTWh * 1_000_000 / (365 * 24)
}

func (r ElectricityReference) Source() string {
	return electricityReferenceSource
}

var electricityReferences = map[string]ElectricityReference{
	"AL": {Country: "Albania", Year: 2024, PopulationMillions: 2.788, DemandTWh: 8.67, DemandMWhPerCapita: 3.11},
	"AT": {Country: "Austria", Year: 2024, PopulationMillions: 9.121, DemandTWh: 71.69, DemandMWhPerCapita: 7.86},
	"BA": {Country: "Bosnia and Herzegovina", Year: 2024, PopulationMillions: 3.163, DemandTWh: 13.03, DemandMWhPerCapita: 4.12},
	"BE": {Country: "Belgium", Year: 2024, PopulationMillions: 11.732, DemandTWh: 86.35, DemandMWhPerCapita: 7.36},
	"BG": {Country: "Bulgaria", Year: 2024, PopulationMillions: 6.759, DemandTWh: 37.31, DemandMWhPerCapita: 5.52},
	"CH": {Country: "Switzerland", Year: 2024, PopulationMillions: 8.922, DemandTWh: 63.97, DemandMWhPerCapita: 7.17},
	"CY": {Country: "Cyprus", Year: 2024, PopulationMillions: 1.359, DemandTWh: 5.79, DemandMWhPerCapita: 4.26},
	"CZ": {Country: "Czechia", Year: 2024, PopulationMillions: 10.739, DemandTWh: 66.15, DemandMWhPerCapita: 6.16},
	"DE": {Country: "Germany", Year: 2024, PopulationMillions: 84.508, DemandTWh: 522.26, DemandMWhPerCapita: 6.18},
	"DK": {Country: "Denmark", Year: 2024, PopulationMillions: 5.974, DemandTWh: 38.77, DemandMWhPerCapita: 6.49},
	"EE": {Country: "Estonia", Year: 2024, PopulationMillions: 1.360, DemandTWh: 9.18, DemandMWhPerCapita: 6.75},
	"ES": {Country: "Spain", Year: 2024, PopulationMillions: 47.913, DemandTWh: 270.71, DemandMWhPerCapita: 5.65},
	"FI": {Country: "Finland", Year: 2024, PopulationMillions: 5.618, DemandTWh: 85.73, DemandMWhPerCapita: 15.26},
	"FR": {Country: "France", Year: 2024, PopulationMillions: 66.571, DemandTWh: 471.99, DemandMWhPerCapita: 7.09},
	"GR": {Country: "Greece", Year: 2024, PopulationMillions: 10.048, DemandTWh: 56.77, DemandMWhPerCapita: 5.65},
	"HR": {Country: "Croatia", Year: 2024, PopulationMillions: 3.876, DemandTWh: 19.65, DemandMWhPerCapita: 5.07},
	"HU": {Country: "Hungary", Year: 2024, PopulationMillions: 9.669, DemandTWh: 48.73, DemandMWhPerCapita: 5.04},
	"IE": {Country: "Ireland", Year: 2024, PopulationMillions: 5.253, DemandTWh: 36.14, DemandMWhPerCapita: 6.88},
	"IS": {Country: "Iceland", Year: 2024, PopulationMillions: 0.393, DemandTWh: 19.05, DemandMWhPerCapita: 48.42},
	"IT": {Country: "Italy", Year: 2024, PopulationMillions: 59.322, DemandTWh: 318.56, DemandMWhPerCapita: 5.37},
	"LT": {Country: "Lithuania", Year: 2024, PopulationMillions: 2.861, DemandTWh: 12.79, DemandMWhPerCapita: 4.47},
	"LU": {Country: "Luxembourg", Year: 2024, PopulationMillions: 0.673, DemandTWh: 6.76, DemandMWhPerCapita: 10.04},
	"LV": {Country: "Latvia", Year: 2024, PopulationMillions: 1.873, DemandTWh: 7.40, DemandMWhPerCapita: 3.95},
	"ME": {Country: "Montenegro", Year: 2024, PopulationMillions: 0.638, DemandTWh: 3.37, DemandMWhPerCapita: 5.28},
	"MK": {Country: "North Macedonia", Year: 2024, PopulationMillions: 1.823, DemandTWh: 7.02, DemandMWhPerCapita: 3.85},
	"MT": {Country: "Malta", Year: 2024, PopulationMillions: 0.539, DemandTWh: 3.16, DemandMWhPerCapita: 5.86},
	"NL": {Country: "Netherlands", Year: 2024, PopulationMillions: 18.227, DemandTWh: 118.11, DemandMWhPerCapita: 6.48},
	"NO": {Country: "Norway", Year: 2024, PopulationMillions: 5.576, DemandTWh: 138.67, DemandMWhPerCapita: 24.87},
	"PL": {Country: "Poland", Year: 2024, PopulationMillions: 38.515, DemandTWh: 174.09, DemandMWhPerCapita: 4.52},
	"PT": {Country: "Portugal", Year: 2024, PopulationMillions: 10.422, DemandTWh: 57.74, DemandMWhPerCapita: 5.54},
	"RO": {Country: "Romania", Year: 2024, PopulationMillions: 19.003, DemandTWh: 55.68, DemandMWhPerCapita: 2.93},
	"RS": {Country: "Serbia", Year: 2024, PopulationMillions: 6.741, DemandTWh: 37.48, DemandMWhPerCapita: 5.56},
	"SE": {Country: "Sweden", Year: 2024, PopulationMillions: 10.605, DemandTWh: 138.71, DemandMWhPerCapita: 13.08},
	"SI": {Country: "Slovenia", Year: 2024, PopulationMillions: 2.119, DemandTWh: 14.30, DemandMWhPerCapita: 6.75},
	"SK": {Country: "Slovakia", Year: 2024, PopulationMillions: 5.501, DemandTWh: 26.57, DemandMWhPerCapita: 4.83},
	"TR": {Country: "Türkiye", Year: 2024, PopulationMillions: 87.465, DemandTWh: 340.24, DemandMWhPerCapita: 3.89},
	"UA": {Country: "Ukraine", Year: 2022, PopulationMillions: 41.033, DemandTWh: 111.61, DemandMWhPerCapita: 2.72},
	"GB": {Country: "United Kingdom", Year: 2024, PopulationMillions: 69.139, DemandTWh: 317.35, DemandMWhPerCapita: 4.59},
	"XK": {Country: "Kosovo", Year: 2024, PopulationMillions: 1.684, DemandTWh: 7.29, DemandMWhPerCapita: 4.33},
}

func Electricity(code string) (ElectricityReference, bool) {
	r, ok := electricityReferences[strings.ToUpper(strings.TrimSpace(code))]
	return r, ok
}

func ElectricityCount() int {
	return len(electricityReferences)
}
