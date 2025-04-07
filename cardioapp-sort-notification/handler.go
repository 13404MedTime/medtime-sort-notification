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
