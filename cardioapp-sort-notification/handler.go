package function

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"
)

const urlConst = "https://api.admin.u-code.io"
// Handle a serverless request
func Handle(req []byte) string {
	var (
		response Response
		request  NewRequestBody
	)

	// Unmarshal the request body
	err := json.Unmarshal(req, &request)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while unmarshalling request"}
		response.Status = "error"
		responseByte, _ := json.Marshal(response)
		return string(responseByte)
	}

	// Validate app_id field in request data
	if request.Data["app_id"] == nil {
		response.Data = map[string]interface{}{"message": "App id required"}
		response.Status = "error"
		responseByte, _ := json.Marshal(response)
		return string(responseByte)
	}
	appId := request.Data["app_id"].(string)

	// Prepare the request for fetching notifications
	var tableSlug = "notifications"
	offset := request.Data["offset"].(float64)
	limit := request.Data["limit"].(float64)

	getListObjectRequest := Request{
		Data: map[string]interface{}{
			"client_id": request.Data["client_id"].(string),
			"time_take": map[string]interface{}{
				"$lte": time.Now(),
			},
		},
	}
	url := fmt.Sprintf("%v/v1/object/get-list/notifications", urlConst)
	res, err, response := GetListObject(url, tableSlug, appId, getListObjectRequest)
	if err != nil {
		responseByte, _ := json.Marshal(response)
		return string(responseByte)
	}

	// Sort the notifications based on time_take in descending order
	sort.Slice(res.Data.Data.Response, func(i, j int) bool {
		timeI, errI := time.Parse(time.RFC3339Nano, res.Data.Data.Response[i]["time_take"].(string))
		timeJ, errJ := time.Parse(time.RFC3339Nano, res.Data.Data.Response[j]["time_take"].(string))

		if errI != nil || errJ != nil {
			return false
		}

		return timeI.After(timeJ)
	})

	// Return a subset of the sorted notifications
	response.Data = map[string]interface{}{"data": getSubset(res.Data.Data.Response, offset, limit)}
	response.Status = "done" // Status indicating success
	responseByte, _ := json.Marshal(response)

	return string(responseByte)
}

func getSubset(data []map[string]interface{}, offset, limit float64) []map[string]interface{} {
	if offset < 0 {
		offset = 0
	}
	if int(offset) >= len(data) {
		return make([]map[string]interface{}, 0)
	}

	endIndex := int(offset) + int(limit)
	if endIndex > len(data) {
		endIndex = len(data)
	}

	return data[int(offset):endIndex]
}

func GetListObject(url, tableSlug, appId string, request Request) (GetListClientApiResponse, error, Response) {
	response := Response{}

	// Make the API request to get the list of notifications
	getListResponseInByte, err := DoRequest(url, "POST", request, appId)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while getting single object"}
		response.Status = "error"
		return GetListClientApiResponse{}, errors.New("error"), response
	}

	// Unmarshal the response into the GetListClientApiResponse structure
	var getListObject GetListClientApiResponse
	err = json.Unmarshal(getListResponseInByte, &getListObject)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while unmarshalling get list object"}
		response.Status = "error"
		return GetListClientApiResponse{}, errors.New("error"), response
	}
	return getListObject, nil, response
}
