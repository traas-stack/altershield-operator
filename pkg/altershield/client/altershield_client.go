package client

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/klog/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultScheme = "http"

	HttpHeaderPlatform  = "X-Altershield-Platform"
	HttpHeaderTimestamp = "X-Altershield-Timestamp"
	HttpHeaderSign      = "X-Altershield-Sign"

	ApiVersion                     = "v1"
	OpenApiFormat                  = "/openapi/%s/exe/%s" // uri format: /openapi/{API version}/exe/{action}
	ActionSubmitChangeExecOrder    = "submitChangeExecOrder"
	ActionSubmitChangeExecOrderStartNotify = "submitChangeExecOrderStartNotify"
	ActionSubmitChangeBatchStartNotify  = "submitChangeExecBatchStartNotify"
	ActionSubmitChangeFinishNotify = "submitChangeFinishNotify"
)

type ChangeCheckType string

const (
	ChangeCheckTypeBatch ChangeCheckType = "CHANGE_BATCH"
	ChangeCheckTypeOrder ChangeCheckType = "CHANGE_ORDER"
)

type DefenseStageType string

const (
	DefenseStageTypePre  DefenseStageType = "PRE"
	DefenseStageTypePost DefenseStageType = "POST"
)

type DefenseVerdictType string

const (
	DefenseVerdictPass DefenseVerdictType = "pass"
	DefenseVerdictFail DefenseVerdictType = "fail"
)

var DefaultChangeContents = []ChangeContent{{
	ContentType: ChangeContentType{
		TypeName: "pass.pod",
	},
	InstanceName: ""}}

const (
	MaxRetryCount = 3
	RetryInterval = time.Second * 1
)

type ChangeCheckCallbackWrapperRequest struct {
	ChangeCheckType ChangeCheckType            `json:"changeCheckType"`
	CallbackRequest ChangeCheckCallbackRequest `json:"callbackRequest"`
}

type ChangeCheckCallbackRequest struct {
	NodeID           string             `json:"nodeId"`
	ChangeSceneKey   string             `json:"changeSceneKey"`
	BizExecOrderID   string             `json:"bizExecOrderId"`
	Verdict          ChangeCheckVerdict `json:"verdict"`
	DefenseStageEnum DefenseStageType   `json:"defenseStageEnum"`
}

type ChangeCheckVerdict struct {
	Verdict DefenseVerdictType `json:"verdict"`
	Msg     string             `json:"msg"`
	NodeId  string             `json:"nodeId"`
}

// sign for your request
func sign(currentTime int64, content string) string {
	strToSign := fmt.Sprintf("%d%s&token=%s", currentTime, content, DefaultIdentity)
	hashed := sha256.Sum256([]byte(strToSign))
	signature := base64.URLEncoding.EncodeToString(hashed[:])
	return strings.ToUpper(signature)
}

func buildPath(action string) string {
	return fmt.Sprintf(OpenApiFormat, ApiVersion, action)
}

type AltershieldClient struct {
	endpoint string
	httpClient *http.Client
}

func NewAltershieldClient(endpoint string) *AltershieldClient {
	return &AltershieldClient{
		endpoint: endpoint,
		httpClient: &http.Client{},
	}
}

func (client *AltershieldClient) String() string {
	if client == nil {
		return "nil"
	}
	return client.endpoint
}

func (client *AltershieldClient) doPost(action string, payload interface{}) (*AltershieldResponse, error) {
	// build url
	u := url.URL{
		Host:   client.endpoint,
		Path:   buildPath(action),
		Scheme: DefaultScheme,
	}
	reqURL := u.String()

	payloadRaw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}
	// build request
	req, err := http.NewRequest("POST", reqURL, bytes.NewReader(payloadRaw))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}
	currentTime := time.Now()
	req.Header.Set(HttpHeaderPlatform, DefaultPlatform)
	req.Header.Set(HttpHeaderTimestamp, strconv.FormatInt(currentTime.Unix(), 10))
	req.Header.Set(HttpHeaderSign, sign(currentTime.Unix(), string(payloadRaw)))
	req.Header.Set("Content-Type", "application/json")

	var resp AltershieldResponse
	var httpResponse *http.Response
	for r := 0;; {
		httpResponse, err = client.httpClient.Do(req)
		if err != nil {
			if r <= MaxRetryCount {
				klog.Errorf("doPost execute error, retry after %d seconds, retry count: %d, url: %v", RetryInterval, r, reqURL)
				r++
			} else {
				return nil, fmt.Errorf("failed to request altershield: %v", err)
			}
		}
		break
	}
	if (httpResponse.StatusCode < http.StatusOK) || (httpResponse.StatusCode >= http.StatusMultipleChoices) {
		return nil, fmt.Errorf("failed to request %v, code %v, message %v",
			action, httpResponse.StatusCode, httpResponse.Status)
	}
	defer httpResponse.Body.Close()
	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %v", err)
	}

	return &resp, nil
}

func (client *AltershieldClient) SubmitChangeExecOrder(request *SubmitChangeExecOrderRequest) (result *SubmitChangeExecOrderResponse, err error) {
	resp, err := client.doPost(ActionSubmitChangeExecOrder, request)
	if err != nil {
		return nil, fmt.Errorf("failed to submit change exec order: %v", err)
	}
	data, err := json.Marshal(resp.Domain)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %v", err)
	}
	result = &SubmitChangeExecOrderResponse{}
	if err = json.Unmarshal(data, result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %v", err)
	}
	return result, nil
}

func (client *AltershieldClient) SubmitChangeExecOrderStartNotify(request *SubmitChangeExecOrderStartNotifyRequest) (
	result *SubmitChangeExecOrderStartNotifyResponse, err error) {
	resp, err := client.doPost(ActionSubmitChangeExecOrderStartNotify, request)
	if err != nil {
		return nil, fmt.Errorf("failed to submit change exec order start notify: %v", err)
	}
	data, err := json.Marshal(resp.Domain)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %v", err)
	}
	result = &SubmitChangeExecOrderStartNotifyResponse{}
	if err = json.Unmarshal(data, result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %v", err)
	}
	return result, nil
}

func (client *AltershieldClient) SubmitChangeExecBatchStartNotify(request *SubmitChangeExecBatchStartNotifyRequest) (
	result *SubmitChangeExecBatchStartNotifyResponse, err error) {
	resp, err := client.doPost(ActionSubmitChangeBatchStartNotify, request)
	if err != nil {
		return nil, fmt.Errorf("failed to submit change exec batch start notify: %v", err)
	}
	data, err := json.Marshal(resp.Domain)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %v", err)
	}
	result = &SubmitChangeExecBatchStartNotifyResponse{}
	if err = json.Unmarshal(data, result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %v", err)
	}
	return result, nil
}

func (client *AltershieldClient) SubmitChangeFinishNotify(request *SubmitChangeFinishNotifyRequest) (result *SubmitChangeFinishNotifyResponse, err error) {
	resp, err := client.doPost(ActionSubmitChangeFinishNotify, request)
	if err != nil {
		return nil, fmt.Errorf("failed to submit change finish notify: %v", err)
	}
	data, err := json.Marshal(resp.Domain)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %v", err)
	}
	result = &SubmitChangeFinishNotifyResponse{}
	if err = json.Unmarshal(data, result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %v", err)
	}
	return result, nil
}

func (client *AltershieldClient) SubmitChangeBatchPostCheck(request *SubmitChangeBatchPostCheckRequest) (*SubmitChangeBatchPostCheckResponse, error) {
	return nil, nil
}

func (client *AltershieldClient) QueryChangeBatchPostCheck(request *QueryChangeBatchPostCheckRequest) (*QueryChangeBatchPostCheckResponse, error) {
	return nil, nil
}

func (client *AltershieldClient) ChangeBatchPreCheck(request *ChangeBatchPreCheckRequest) (*ChangeBatchPreCheckResponse, error) {
	return nil, nil
}
