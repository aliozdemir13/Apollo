// Package weather fetches current conditions from Open-Meteo, a free API that
// requires no key. It also geocodes place names and can auto-detect the user's
// location by IP as a fallback.
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"time"
)

// Service performs weather lookups.
type Service struct {
	client *http.Client
}

// New returns a Service with a sane HTTP timeout.
func New() *Service {
	return &Service{client: &http.Client{Timeout: 12 * time.Second}}
}

var (
	geocodeBase  = "https://geocoding-api.open-meteo.com/v1/search"
	ipapiURL     = "https://ipapi.co/json/"
	forecastBase = "https://api.open-meteo.com/v1/forecast"
)

// Geocode resolves a place name to coordinates via Open-Meteo's geocoder.
func (s *Service) Geocode(ctx context.Context, name string) (GeoResult, error) {
	q := url.Values{}
	q.Set("count", "1")
	q.Set("language", "en")
	q.Set("format", "json")
	q.Set("name", name)
	u := geocodeBase + "?" + q.Encode()

	slog.Info("API call", "url", u)

	var out struct {
		Results []GeoResult `json:"results"`
	}
	if err := s.getJSON(ctx, u, &out); err != nil {
		// error for geolocation capturing api logged in the getJSON
		return GeoResult{}, err
	}
	if len(out.Results) == 0 {
		slog.Error("no location found", "name", name)
		return GeoResult{}, fmt.Errorf("no location found for %q", name)
	}
	return out.Results[0], nil
}

// DetectLocation guesses the user's location from their IP address. Used only
// when no location has been configured.
func (s *Service) DetectLocation(ctx context.Context) (GeoResult, error) {
	var out struct {
		City      string  `json:"city"`
		Region    string  `json:"region"`
		Country   string  `json:"country_name"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Error     bool    `json:"error"`
	}
	if err := s.getJSON(ctx, ipapiURL, &out); err != nil {
		// error for location definition via ip address logged in the getJSON
		return GeoResult{}, err
	}
	if out.Error || (out.Latitude == 0 && out.Longitude == 0) {
		return GeoResult{}, fmt.Errorf("could not detect location by IP")
	}
	return GeoResult{
		Name:      out.City,
		Admin1:    out.Region,
		Country:   out.Country,
		Latitude:  out.Latitude,
		Longitude: out.Longitude,
	}, nil
}

// CurrentWeather fetches present conditions for the coordinates. units is "celsius"
// or "fahrenheit"; locationName is used purely for display.
func (s *Service) CurrentWeather(ctx context.Context, lat, lon float64, units, locationName string) (Data, error) {
	tempUnit := "celsius"
	unitLabel := "C"
	if units == "fahrenheit" {
		tempUnit = "fahrenheit"
		unitLabel = "F"
	}

	q := url.Values{}
	q.Set("latitude", fmt.Sprintf("%.4f", lat))
	q.Set("longitude", fmt.Sprintf("%.4f", lon))
	q.Set("current", "temperature_2m,relative_humidity_2m,apparent_temperature,is_day,weather_code,wind_speed_10m")
	q.Set("temperature_unit", tempUnit)
	q.Set("wind_speed_unit", "kmh")
	u := forecastBase + "?" + q.Encode()

	slog.Info("API call", "url", u)

	var out CurrentWeather
	if err := s.getJSON(ctx, u, &out); err != nil {
		// error for weather api logged in the getJSON
		return Data{}, err
	}

	label, desc := DescribeCode(out.Current.WeatherCode)
	return Data{
		Location:    locationName,
		Label:       label,
		Description: desc,
		Temp:        math.Round(out.Current.Temperature),
		Unit:        unitLabel,
		FeelsLike:   math.Round(out.Current.ApparentTemp),
		Humidity:    int(out.Current.Humidity + 0.5),
		WindSpeed:   math.Round(out.Current.WindSpeed),
		Code:        out.Current.WeatherCode,
		IsDay:       out.Current.IsDay == 1,
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}, nil
}

// single API handler for the app
func (s *Service) getJSON(ctx context.Context, u string, v interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		slog.Error("Request composition failed", "url", u, "error", err)
		return err
	}
	req.Header.Set("User-Agent", "apollo-widget/1.0") // to avoid auto blocked by system provider
	resp, err := s.client.Do(req)
	if err != nil {
		slog.Error("API call failed", "url", u, "error", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.Error("Unexpected response", "url", u, "statusCode", resp.StatusCode, "response", resp)
		return fmt.Errorf("request to %s failed: %s", u, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}
