package provider

import (
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/model"
)

var tagRE = regexp.MustCompile(`(?s)<[^>]+>`)
var wsRE = regexp.MustCompile(`\s+`)

func htmlText(b []byte) string {
	s := tagRE.ReplaceAllString(string(b), " ")
	s = html.UnescapeString(s)
	return strings.TrimSpace(wsRE.ReplaceAllString(s, " "))
}
func number(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")
	if strings.Count(s, ",") == 1 && strings.Count(s, ".") == 0 {
		s = strings.ReplaceAll(s, ",", ".")
	} else {
		s = strings.ReplaceAll(s, ",", "")
	}
	v, e := strconv.ParseFloat(s, 64)
	return v, e == nil
}
func f64(v float64) *float64 { return &v }
func obs(key, domain, label string, value float64, unit, source, url string, q float64, at time.Time) model.Observation {
	return model.Observation{Key: key, Domain: domain, Label: label, Value: f64(value), Unit: unit, Source: source, SourceURL: url, Quality: q, ObservedAt: at, RetrievedAt: time.Now().UTC()}
}
func textObs(key, domain, label, text, source, url string, q float64, at time.Time) model.Observation {
	return model.Observation{Key: key, Domain: domain, Label: label, Text: text, Source: source, SourceURL: url, Quality: q, ObservedAt: at, RetrievedAt: time.Now().UTC()}
}
