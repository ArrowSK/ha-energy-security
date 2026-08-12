package country

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed data/countries.json
var fs embed.FS

type Profile struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	EnergyCharts string `json:"energy_charts,omitempty"`
	ENTSOEEIC    string `json:"entsoe_eic,omitempty"`
	Eurostat     bool   `json:"eurostat"`
	HungaryGas   bool   `json:"hungary_gas,omitempty"`
	HungaryWater bool   `json:"hungary_water,omitempty"`
	Support      string `json:"support"`
}

var profiles map[string]Profile

func init() {
	b, err := fs.ReadFile("data/countries.json")
	if err != nil {
		panic(err)
	}
	var list []Profile
	if err := json.Unmarshal(b, &list); err != nil {
		panic(err)
	}
	profiles = make(map[string]Profile, len(list))
	for _, p := range list {
		profiles[strings.ToUpper(p.Code)] = p
	}
}

func Get(code string) (Profile, bool) { p, ok := profiles[strings.ToUpper(code)]; return p, ok }
func Resolve(code string) Profile {
	if p, ok := Get(code); ok {
		return p
	}
	c := strings.ToUpper(code)
	return Profile{Code: c, Name: c, Support: "limited"}
}
func All() []Profile {
	out := make([]Profile, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, p)
	}
	return out
}
func Validate() error {
	for k, p := range profiles {
		if len(k) != 2 || p.Code == "" || p.Name == "" {
			return fmt.Errorf("invalid profile %q", k)
		}
	}
	return nil
}
