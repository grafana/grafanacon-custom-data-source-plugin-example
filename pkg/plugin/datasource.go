package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/my-test-org/test/pkg/models"
)

// Make sure Datasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler interfaces. Plugin should not implement all these
// interfaces - only those which are required for a particular task.
var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

// NewDatasource creates a new datasource instance.
func NewDatasource(_ context.Context, _ backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &Datasource{}, nil
}

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct{}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *Datasource) Dispose() {
	// Clean up datasource instance resources.
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := d.query(ctx, req.PluginContext, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

type queryModel struct {
	QueryText string `json:"queryText"`
}

type RoadWeatherData struct {
	DateTime               string `json:"datetime"`
	RoadSurfaceTemperature string `json:"roadsurfacetemperature"`
	AirTemperature         string `json:"airtemperature"`
}

func (d *Datasource) query(_ context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	var response backend.DataResponse

	// Unmarshal the JSON into our queryModel.
	var qm queryModel

	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err.Error()))
	}

	backend.Logger.Debug("Query text", "queryText", qm.QueryText)
	// Load the plugin settings
	config, err := models.LoadPluginSettings(*pCtx.DataSourceInstanceSettings)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("failed to load plugin settings: %v", err.Error()))
	}

	// Build the URL with config settings
	url := config.Domain + "/resource/" + config.Resource + ".json" + "?$$app_token=" + config.Secrets.AppToken

	if qm.QueryText != "" {
		url += "&" + qm.QueryText
	}

	backend.Logger.Debug("Making request to", "url", url)

	// Make the request
	resp, err := http.Get(url)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("failed to make request: %v", err.Error()))
	}

	// Close the response body after this function is done executing
	defer resp.Body.Close()

	// Decode the response body into a RoadWeatherData struct
	var roadWeatherData []RoadWeatherData
	err = json.NewDecoder(resp.Body).Decode(&roadWeatherData)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("failed to decode response body: %v", err.Error()))
	}

	// Create slices to store the data as columnar data
	times := make([]time.Time, len(roadWeatherData))
	roadSurfaceTemps := make([]float64, len(roadWeatherData))
	airTemps := make([]float64, len(roadWeatherData))

	for i, roadWeather := range roadWeatherData {
		times[i], err = time.Parse("2006-01-02T15:04:05.000", roadWeather.DateTime)
		if err != nil {
			return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("failed to parse datetime: %v", err.Error()))
		}
		roadSurfaceTemps[i], err = strconv.ParseFloat(roadWeather.RoadSurfaceTemperature, 64)
		if err != nil {
			return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("failed to parse road surface temp: %v", err.Error()))
		}
		airTemps[i], err = strconv.ParseFloat(roadWeather.AirTemperature, 64)
		if err != nil {
			return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("failed to parse air temp: %v", err.Error()))
		}
	}

	// create data frame response.
	// For an overview on data frames and how grafana handles them:
	// https://grafana.com/developers/plugin-tools/introduction/data-frames
	frame := data.NewFrame("response")

	// add fields.
	frame.Fields = append(frame.Fields,
		data.NewField("time", nil, times),
		data.NewField("road temperatures", nil, roadSurfaceTemps),
		data.NewField("air temperatures", nil, airTemps),
	)

	// add the frames to the response.
	response.Frames = append(response.Frames, frame)

	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(_ context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	res := &backend.CheckHealthResult{}
	config, err := models.LoadPluginSettings(*req.PluginContext.DataSourceInstanceSettings)

	if err != nil {
		res.Status = backend.HealthStatusError
		res.Message = "Unable to load settings"
		return res, nil
	}

	if config.Secrets.AppToken == "" {
		res.Status = backend.HealthStatusError
		res.Message = "App token is missing"
		return res, nil
	}

	// Build the URL with the API key
	url := config.Domain + "/resource/" + config.Resource + ".json" + "?$$app_token=" + config.Secrets.AppToken

	// Log the URL for debugging purposes
	backend.Logger.Debug("Testing connection to", "url", url)

	// Make a request to the URL
	resp, err := http.Get(url)

	// Check if the request failed and return an error if it did
	if err != nil {
		backend.Logger.Error("Failed to make request", "error", err)
		return nil, err
	}

	// Check if the response status is not 200 and return an error if it's not
	if resp.StatusCode != 200 {
		backend.Logger.Error("Request failed with status code", "status", resp.StatusCode)
		return nil, fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}

	// Close the response body
	defer resp.Body.Close()

	// If we got this far, the request was successful
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Data source is working",
	}, nil
}
