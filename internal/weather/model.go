// Package weather fetches current conditions from Open-Meteo, a free API that
// requires no key. It also geocodes place names and can auto-detect the user's
// location by IP as a fallback.
package weather

// Data is the current-conditions payload returned to the frontend.
type Data struct {
	Location    string  `json:"location"`    // resolved place name
	Label       string  `json:"label"`       // short condition, e.g. "FAIR"
	Description string  `json:"description"` // longer condition, e.g. "Partly cloudy"
	Temp        float64 `json:"temp"`        // in the requested unit, rounded
	Unit        string  `json:"unit"`        // "C" | "F"
	FeelsLike   float64 `json:"feelsLike"`   // apparent temperature
	Humidity    int     `json:"humidity"`    // %
	WindSpeed   float64 `json:"windSpeed"`   // km/h
	Code        int     `json:"code"`        // WMO weather code
	IsDay       bool    `json:"isDay"`       // for icon day/night
	UpdatedAt   string  `json:"updatedAt"`   // RFC3339
}

// GeoResult is a single geocoding match.
type GeoResult struct {
	Name      string  `json:"name"`
	Country   string  `json:"country"`
	Admin1    string  `json:"admin1"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type CurrentWeatherDetails struct {
	Temperature  float64 `json:"temperature_2m"`
	Humidity     float64 `json:"relative_humidity_2m"`
	ApparentTemp float64 `json:"apparent_temperature"`
	IsDay        int     `json:"is_day"`
	WeatherCode  int     `json:"weather_code"`
	WindSpeed    float64 `json:"wind_speed_10m"`
}

type CurrentWeather struct {
	Current CurrentWeatherDetails `json:"current"`
}

// describeCode maps a WMO weather code to a short LCD label and a longer
// description. The short label mimics the screenshot's "FAIR" styling.
func DescribeCode(code int) (short, long string) {
	switch code {
	case 0:
		return "CLEAR", "Clear sky"
	case 1:
		return "FAIR", "Mainly clear"
	case 2:
		return "PART", "Partly cloudy"
	case 3:
		return "CLOUD", "Overcast"
	case 45, 48:
		return "FOG", "Fog"
	case 51, 53, 55:
		return "DRIZZLE", "Drizzle"
	case 56, 57:
		return "DRIZZLE", "Freezing drizzle"
	case 61, 63, 65:
		return "RAIN", "Rain"
	case 66, 67:
		return "RAIN", "Freezing rain"
	case 71, 73, 75, 77:
		return "SNOW", "Snow"
	case 80, 81, 82:
		return "SHOWER", "Rain showers"
	case 85, 86:
		return "SNOW", "Snow showers"
	case 95:
		return "STORM", "Thunderstorm"
	case 96, 99:
		return "STORM", "Thunderstorm w/ hail"
	default:
		return "----", "Unknown"
	}
}
