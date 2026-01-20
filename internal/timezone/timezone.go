package timezone

import (
	"strings"
	"time"
)

var (
	WIB  *time.Location // UTC+7 - Western Indonesia (Jakarta, Surabaya)
	WITA *time.Location // UTC+8 - Central Indonesia (Bali, Makassar)
	WIT  *time.Location // UTC+9 - Eastern Indonesia (Papua)
)

func init() {
	WIB = time.FixedZone("WIB", 7*60*60)
	WITA = time.FixedZone("WITA", 8*60*60)
	WIT = time.FixedZone("WIT", 9*60*60)
}

var airportTimezones = map[string]string{
	// WIB (UTC+7) - Western Indonesia
	"CGK": "WIB", // Jakarta - Soekarno-Hatta
	"HLP": "WIB", // Jakarta - Halim Perdanakusuma
	"BDO": "WIB", // Bandung - Husein Sastranegara
	"SUB": "WIB", // Surabaya - Juanda
	"SRG": "WIB", // Semarang - Ahmad Yani
	"JOG": "WIB", // Yogyakarta - Adisucipto
	"SOC": "WIB", // Solo - Adisumarmo
	"PLM": "WIB", // Palembang - Sultan Mahmud Badaruddin II
	"PNK": "WIB", // Pontianak - Supadio
	"BTH": "WIB", // Batam - Hang Nadim
	"PKU": "WIB", // Pekanbaru - Sultan Syarif Kasim II
	"PDG": "WIB", // Padang - Minangkabau
	"KNO": "WIB", // Medan - Kualanamu
	"BTJ": "WIB", // Banda Aceh - Sultan Iskandar Muda
	"TNJ": "WIB", // Tanjung Pinang - Raja Haji Fisabilillah

	// WITA (UTC+8) - Central Indonesia
	"DPS": "WITA", // Bali - Ngurah Rai
	"LOP": "WITA", // Lombok - Lombok International
	"UPG": "WITA", // Makassar - Sultan Hasanuddin
	"BPN": "WITA", // Balikpapan - Sultan Aji Muhammad Sulaiman
	"MDC": "WITA", // Manado - Sam Ratulangi
	"KDI": "WITA", // Kendari - Haluoleo
	"PLW": "WITA", // Palu - Mutiara SIS Al-Jufri
	"TRK": "WITA", // Tarakan - Juwata

	// WIT (UTC+9) - Eastern Indonesia
	"DJJ": "WIT", // Jayapura - Sentani
	"TIM": "WIT", // Timika - Mozes Kilangin
	"BIK": "WIT", // Biak - Frans Kaisiepo
	"MKQ": "WIT", // Merauke - Mopah
	"SOQ": "WIT", // Sorong - Domine Eduard Osok
	"AMQ": "WIT", // Ambon - Pattimura
}

func GetTimezoneByAirport(code string) string {
	code = strings.ToUpper(code)
	if tz, ok := airportTimezones[code]; ok {
		return tz
	}
	return "WIB"
}

func GetLocationByAirport(code string) *time.Location {
	tz := GetTimezoneByAirport(code)
	switch tz {
	case "WITA":
		return WITA
	case "WIT":
		return WIT
	default:
		return WIB
	}
}

func GetLocationByName(name string) *time.Location {
	switch strings.ToUpper(name) {
	case "WITA", "UTC+8":
		return WITA
	case "WIT", "UTC+9":
		return WIT
	case "WIB", "UTC+7":
		return WIB
	default:
		if loc, err := time.LoadLocation(name); err == nil {
			return loc
		}
		return WIB
	}
}

func ParseTimeWithOffset(timeStr string, tzName string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05+07:00",
		"2006-01-02T15:04:05-0700", // Without colon
		"2006-01-02T15:04:05+0700", // Without colon
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	if tzName != "" {
		loc := GetLocationByName(tzName)
		simpleFormats := []string{
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
			"2006-01-02T15:04",
			"2006-01-02 15:04",
		}
		for _, format := range simpleFormats {
			if t, err := time.ParseInLocation(format, timeStr, loc); err == nil {
				return t, nil
			}
		}
	}

	return time.Time{}, &time.ParseError{
		Value:   timeStr,
		Message: "unable to parse time string",
	}
}

func ConvertToTimezone(t time.Time, airportCode string) time.Time {
	loc := GetLocationByAirport(airportCode)
	return t.In(loc)
}
