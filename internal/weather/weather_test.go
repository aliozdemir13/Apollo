package weather

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRound(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{"zero", 0, 0},
		{"round up", 1.5, 2},
		{"round down", 1.4, 1},
		{"negative round", -1.5, -2},
		{"negative small", -1.4, -1},
		{"already integer", 7, 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := math.Round(tt.in); got != tt.want {
				t.Errorf("round(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestDescribeCode(t *testing.T) {
	tests := []struct {
		code      int
		wantShort string
		wantLong  string
	}{
		{0, "CLEAR", "Clear sky"},
		{1, "FAIR", "Mainly clear"},
		{2, "PART", "Partly cloudy"},
		{3, "CLOUD", "Overcast"},
		{45, "FOG", "Fog"},
		{48, "FOG", "Fog"},
		{51, "DRIZZLE", "Drizzle"},
		{56, "DRIZZLE", "Freezing drizzle"},
		{61, "RAIN", "Rain"},
		{66, "RAIN", "Freezing rain"},
		{71, "SNOW", "Snow"},
		{80, "SHOWER", "Rain showers"},
		{85, "SNOW", "Snow showers"},
		{95, "STORM", "Thunderstorm"},
		{96, "STORM", "Thunderstorm w/ hail"},
		{999, "----", "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.wantShort+"_"+tt.wantLong, func(t *testing.T) {
			short, long := DescribeCode(tt.code)
			if short != tt.wantShort || long != tt.wantLong {
				t.Errorf("DescribeCode(%d) = (%q,%q), want (%q,%q)", tt.code, short, long, tt.wantShort, tt.wantLong)
			}
		})
	}
}

// serve spins up a test server returning status/body and points the given base
// var at it. Returns a restore func.
func serve(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

func TestGeocode(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		badBase  string
		wantErr  bool
		wantName string
	}{
		{name: "ok", status: 200, body: `{"results":[{"name":"Munich","latitude":48.1,"longitude":11.5}]}`, wantName: "Munich"},
		{name: "no results", status: 200, body: `{"results":[]}`, wantErr: true},
		{name: "http error", status: 500, body: ``, wantErr: true},
		{name: "bad json", status: 200, body: `{not json`, wantErr: true},
		{name: "bad base url", badBase: "http://\x7f", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := geocodeBase
			defer func() { geocodeBase = old }()
			if tt.badBase != "" {
				geocodeBase = tt.badBase
			} else {
				srv := serve(t, tt.status, tt.body)
				defer srv.Close()
				geocodeBase = srv.URL
			}
			got, err := New().Geocode(context.Background(), "x")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if !tt.wantErr && got.Name != tt.wantName {
				t.Errorf("name=%q want %q", got.Name, tt.wantName)
			}
		})
	}
}

func TestGetJSONConnError(t *testing.T) {
	// A server that is closed before use makes client.Do fail (connection error),
	// covering getJSON's transport-error branch.
	srv := serve(t, 200, `{}`)
	url := srv.URL
	srv.Close()
	old := geocodeBase
	defer func() { geocodeBase = old }()
	geocodeBase = url
	if _, err := New().Geocode(context.Background(), "x"); err == nil {
		t.Fatal("expected connection error, got nil")
	}
}

func TestDetectLocation(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		badBase string
		wantErr bool
	}{
		{name: "ok", status: 200, body: `{"city":"Berlin","latitude":52.5,"longitude":13.4}`},
		{name: "error flag", status: 200, body: `{"error":true}`, wantErr: true},
		{name: "zero coords", status: 200, body: `{"city":"X","latitude":0,"longitude":0}`, wantErr: true},
		{name: "http error", status: 500, body: ``, wantErr: true},
		{name: "bad base", badBase: "http://\x7f", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := ipapiURL
			defer func() { ipapiURL = old }()
			if tt.badBase != "" {
				ipapiURL = tt.badBase
			} else {
				srv := serve(t, tt.status, tt.body)
				defer srv.Close()
				ipapiURL = srv.URL
			}
			_, err := New().DetectLocation(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestCurrent(t *testing.T) {
	body := `{"current":{"temperature_2m":29.6,"relative_humidity_2m":55.4,"apparent_temperature":31.2,"is_day":1,"weather_code":1,"wind_speed_10m":12.3}}`
	tests := []struct {
		name      string
		units     string
		status    int
		body      string
		badBase   string
		wantErr   bool
		wantUnit  string
		wantTemp  float64
		wantLabel string
		wantDay   bool
	}{
		{name: "celsius", units: "celsius", status: 200, body: body, wantUnit: "C", wantTemp: 30, wantLabel: "FAIR", wantDay: true},
		{name: "fahrenheit", units: "fahrenheit", status: 200, body: body, wantUnit: "F", wantTemp: 30, wantLabel: "FAIR", wantDay: true},
		{name: "http error", units: "celsius", status: 500, body: ``, wantErr: true},
		{name: "bad base", units: "celsius", badBase: "http://\x7f", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := forecastBase
			defer func() { forecastBase = old }()
			if tt.badBase != "" {
				forecastBase = tt.badBase
			} else {
				srv := serve(t, tt.status, tt.body)
				defer srv.Close()
				forecastBase = srv.URL
			}
			got, err := New().CurrentWeather(context.Background(), 1, 2, tt.units, "Here")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.Unit != tt.wantUnit || got.Temp != tt.wantTemp || got.Label != tt.wantLabel || got.IsDay != tt.wantDay {
				t.Errorf("got %+v", got)
			}
			if got.Location != "Here" || got.Humidity != 55 {
				t.Errorf("location/humidity wrong: %+v", got)
			}
			if got.UpdatedAt == "" {
				t.Error("UpdatedAt empty")
			}
		})
	}
}
