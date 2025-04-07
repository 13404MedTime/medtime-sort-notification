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

	err := json.Unmarshal(req, &request)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while unmarshalling request"}
		response.Status = "error"
		responseByte, _ := json.Marshal(response)
		return string(responseByte)
	}
	if request.Data["app_id"] == nil {
		response.Data = map[string]interface{}{"message": "App id required"}
		response.Status = "error"
		responseByte, _ := json.Marshal(response)
		return string(responseByte)
	}
	appId := request.Data["app_id"].(string)

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

	sort.Slice(res.Data.Data.Response, func(i, j int) bool {
		timeI, errI := time.Parse(time.RFC3339Nano, res.Data.Data.Response[i]["time_take"].(string))
		timeJ, errJ := time.Parse(time.RFC3339Nano, res.Data.Data.Response[j]["time_take"].(string))

		if errI != nil || errJ != nil {
			return false
		}

		return timeI.After(timeJ)
	})

	response.Data = map[string]interface{}{"data": getSubset(res.Data.Data.Response, offset, limit)}
	response.Status = "done" //if all will be ok else "error"
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

	getListResponseInByte, err := DoRequest(url, "POST", request, appId)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while getting single object"}
		response.Status = "error"
		return GetListClientApiResponse{}, errors.New("error"), response
	}
	var getListObject GetListClientApiResponse
	err = json.Unmarshal(getListResponseInByte, &getListObject)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while unmarshalling get list object"}
		response.Status = "error"
		return GetListClientApiResponse{}, errors.New("error"), response
	}
	return getListObject, nil, response
}

type Datas struct {
	Data struct {
		Data struct {
			Data map[string]interface{} `json:"data"`
		} `json:"data"`
	} `json:"data"`
}

// ClientApiResponse This is get single api response
type ClientApiResponse struct {
	Data ClientApiData `json:"data"`
}

type ClientApiData struct {
	Data ClientApiResp `json:"data"`
}

type ClientApiResp struct {
	Response map[string]interface{} `json:"response"`
}

type Response struct {
	Status string                 `json:"status"`
	Data   map[string]interface{} `json:"data"`
}

type HttpRequest struct {
	Method  string      `json:"method"`
	Path    string      `json:"path"`
	Headers http.Header `json:"headers"`
	Params  url.Values  `json:"params"`
	Body    []byte      `json:"body"`
}

type AuthData struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

type NewRequestBody struct {
	RequestData HttpRequest            `json:"request_data"`
	Auth        AuthData               `json:"auth"`
	Data        map[string]interface{} `json:"data"`
}
type Request struct {
	Data map[string]interface{} `json:"data"`
}

// GetListClientApiResponse This is get list api response
type GetListClientApiResponse struct {
	Data GetListClientApiData `json:"data"`
}

type GetListClientApiData struct {
	Data GetListClientApiResp `json:"data"`
}

type GetListClientApiResp struct {
	Response []map[string]interface{} `json:"response"`
}

func DoRequest(url string, method string, body interface{}, appId string) ([]byte, error) {
	data, err := json.Marshal(&body)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: time.Duration(5 * time.Second),
	}

	request, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	request.Header.Add("authorization", "API-KEY")
	request.Header.Add("X-API-KEY", appId)

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respByte, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return respByte, nil
}
