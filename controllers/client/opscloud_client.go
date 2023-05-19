package client

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
)

const (
	HttpHeaderPlatformKey          = "X-OpsCloud-Platform"
	HttpHeaderTimestampKey         = "X-OpsCloud-Timestamp"
	HttpHeaderSignKey              = "X-OpsCloud-Sign"
	HttpHeaderTargetTenant         = "X-Tldc-Target-Tenant"
	HttpHeaderTarget               = "X-Tldc-Target-Biz"
	ApiVersion                     = "v1"
	OpenApiFormat                  = "/openapi/%s/exe/%s" // uri format: /openapi/{API version}/exe/{action}
	SubmitChangeExecOrderAction    = "submitChangeExecOrder"
	SubmitChangeStartNotifyAction  = "submitChangeExecBatchStartNotify"
	SubmitChangeFinishNotifyAction = "submitChangeFinishNotify"
)

const (
	Platform       = "cafed"
	Token          = "123"
	TargetTenant   = "cloudbaseapp:crconsole"
	TargetService  = "topscloud-sdk-api"
	OpsCloudDomain = "http://localhost:8080"
	// OpsCloudDomain = "http://topscloud-pool.gzcr60a.cloudrun-uat.local:8080"

)

const (
	OpsCloudChangeCheckTypeEnumBatch = "CHANGE_BATCH"
	DefenseStageEnumPre              = "PRE"
	DefenseStageEnumPost             = "POST"
)

var DefaultChangeContents = []OpsCloudChangeContent{{
	ContentType: OpsCloudChangeContentType{
		TypeName: "pass.pod",
	},
	InstanceName: ""}}

const (
	MaxRetryCount = 3
	RetryInterval = time.Second * 1
)

var logger = utils.NewLogger()

type OpsCloudResult struct {
	Success    bool        `json:"success"`
	ResultCode string      `json:"resultCode"`
	Msg        string      `json:"msg"`
	Domain     interface{} `json:"domain"`
}

type OpsCloudChangeCheckCallbackWrapperRequest struct {
	ChangeCheckType string                             `json:"changeCheckType"`
	CallbackRequest OpsCloudChangeCheckCallbackRequest `json:"callbackRequest"`
}

type OpsCloudChangeCheckCallbackRequest struct {
	NodeId           string                     `json:"nodeId"`
	ChangeSceneKey   string                     `json:"changeSceneKey"`
	BizExecOrderId   string                     `json:"bizExecOrderId"`
	Verdict          OpsCloudChangeCheckVerdict `json:"verdict"`
	DefenseStageEnum string                     `json:"defenseStageEnum"`
}

type OpsCloudChangeCheckVerdict struct {
	Verdict string `json:"verdict"`
	Msg     string `json:"msg"`
	NodeId  string `json:"nodeId"`
}

// OpsCloudChangeExecOrderSubmitRequest submit change execute order request
type OpsCloudChangeExecOrderSubmitRequest struct {
	BizExecOrderId     string                  `json:"bizExecOrderId"`
	Platform           string                  `json:"platform"`
	ChangeSceneKey     string                  `json:"changeSceneKey"`
	ChangeApps         []string                `json:"changeApps"`
	ChangeParamJson    string                  `json:"changeParamJson"`
	ChangePhases       []string                `json:"changePhases"`
	ChangeScenarioCode string                  `json:"changeScenarioCode"`
	ChangeTitle        string                  `json:"changeTitle"`
	ChangeUrl          string                  `json:"changeUrl"`
	Creator            string                  `json:"creator"`
	ChangeContents     []OpsCloudChangeContent `json:"changeContents"`
	TldcTenantCode     string                  `json:"tldcTenantCode"`
}

type OpsCloudChangeContent struct {
	ContentType  OpsCloudChangeContentType `json:"contentType"`
	InstanceName string                    `json:"instanceName"`
}

type OpsCloudChangeContentType struct {
	TypeName string `json:"typeName"`
}

// OpsCloudChangeExecBatchStartNotifyRequest submit batch execute order start request
type OpsCloudChangeExecBatchStartNotifyRequest struct {
	Platform                   string   `json:"platform"`
	BizExecOrderId             string   `json:"bizExecOrderId"`
	ChangeSceneKey             string   `json:"changeSceneKey"`
	BatchNo                    uint64   `json:"batchNo"`
	ChangePhase                string   `json:"changePhase"`
	EffectiveTargetType        string   `json:"effectiveTargetType"`
	EffectiveTargetLocations   []string `json:"effectiveTargetLocations"`
	Executor                   string   `json:"executor"`
	TotalBatchNum              uint64   `json:"totalBatchNum"`
	TotalBatchNumInChangePhase uint64   `json:"totalBatchNumInChangePhase"`
	TldcTenantCode             string   `json:"tldcTenantCode"`
}

// OpsCloudChangeFinishNotifyRequest submit batch execute order finish request
type OpsCloudChangeFinishNotifyRequest struct {
	BizExecOrderId string `json:"bizExecOrderId"`
	ChangeSceneKey string `json:"changeSceneKey"`
	NodeId         string `json:"nodeId"`
	Platform       string `json:"platform"`
	ServiceResult  string `json:"serviceResult"`
	Success        bool   `json:"success"`
	TldcTenantCode string `json:"tldcTenantCode"`
}

// SubmitChangeExecOrder submit a execute order
func SubmitChangeExecOrder(request OpsCloudChangeExecOrderSubmitRequest) (OpsCloudResult, error) {
	return serviceTemplate(request, SubmitChangeExecOrderAction, "SubmitChangeExecOrder")
}

// SubmitChangeStartNotify sync batch order start
func SubmitChangeStartNotify(request OpsCloudChangeExecBatchStartNotifyRequest) (OpsCloudResult, error) {
	return serviceTemplate(request, SubmitChangeStartNotifyAction, "SubmitChangeStartNotify")
}

// SubmitChangeFinishNotify async batch order finish
func SubmitChangeFinishNotify(request OpsCloudChangeFinishNotifyRequest) (OpsCloudResult, error) {
	return serviceTemplate(request, SubmitChangeFinishNotifyAction, "SubmitChangeFinishNotify")
}

func serviceTemplate(request interface{}, action string, method string) (OpsCloudResult, error) {
	startTime := time.Now()
	var result OpsCloudResult
	result, err := doPost(buildUri(action), request)
	if err != nil {
		logger.WithValues("request", request).WithValues("result", result).Error(err, method+" doPost error"+time.Since(startTime).String()) //, request, result
		return result, err
	}
	if !result.Success {
		logger.WithValues("msg", result.Msg).WithValues("ResultCode", result.ResultCode).WithValues("request", request).WithValues("result", result).Error(err, method+" failed"+time.Since(startTime).String()) //, result.Msg, result.ResultCode, request, result
		return result, errors.New(result.Msg)
	} else {
		logger.Info(method + " success" + time.Since(startTime).String()) //, request, result
		return result, nil
	}
}

// sign for your request
func sign(currentTime int64, content string) string {
	strToSign := fmt.Sprintf("%d%s&token=%s", currentTime, content, Token)
	hashed := sha256.Sum256([]byte(strToSign))
	signature := base64.URLEncoding.EncodeToString(hashed[:])
	return strings.ToUpper(signature)
}

func doPost(uri string, request interface{}) (OpsCloudResult, error) {
	bytes, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", uri, strings.NewReader(string(bytes)))
	currentTime := time.Now()

	req.Header.Set(HttpHeaderPlatformKey, Platform)
	req.Header.Set(HttpHeaderTimestampKey, strconv.FormatInt(currentTime.Unix(), 10))
	//req.Header.Set(HttpHeaderTargetTenant, TargetTenant)
	//req.Header.Set(HttpHeaderTarget, TargetService)
	req.Header.Set(HttpHeaderSignKey, sign(currentTime.Unix(), string(bytes)))
	req.Header.Set("Content-Type", "application/json")
	var serviceRes OpsCloudResult
	var resp *http.Response
	var err error

	for r := 0; ; r++ {
		resp, err = (&http.Client{}).Do(req)
		if err != nil && r <= MaxRetryCount {
			logger.WithValues("url", uri).WithValues("request", request).Error(err, fmt.Sprintf("doPost execute error, retry after %d seconds, retry count: %d", RetryInterval, r)) //, uri, request
			time.Sleep(RetryInterval)
		} else if err == nil {
			break
		} else {
			return serviceRes, err
		}
	}
	defer resp.Body.Close()
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return OpsCloudResult{}, err
	}
	err = json.Unmarshal(all, &serviceRes)
	return serviceRes, err
}

func buildUri(action string) string {
	return OpsCloudDomain + fmt.Sprintf(OpenApiFormat, ApiVersion, action)
}

// SubmitChangeExecOrderWeb TODO delete
// SubmitChangeExecOrderWeb 测试client接口
func SubmitChangeExecOrderWeb(c *gin.Context) {
	logger := utils.NewLogger()
	var data OpsCloudChangeExecOrderSubmitRequest
	if err := c.ShouldBindJSON(&data); err != nil {
		logger.Error(err, "OpsCloudChangeExecOrderSubmitRequest:bind json error")
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	result, err := SubmitChangeExecOrder(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func SubmitChangeStartNotifyWeb(c *gin.Context) {
	logger := utils.NewLogger()
	var data OpsCloudChangeExecBatchStartNotifyRequest
	if err := c.ShouldBindJSON(&data); err != nil {
		logger.Error(err, "OpsCloudChangeExecBatchStartNotifyRequest:bind json error")
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	result, err := SubmitChangeStartNotify(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func SubmitChangeFinishNotifyWeb(c *gin.Context) {
	logger := utils.NewLogger()
	var data OpsCloudChangeFinishNotifyRequest
	if err := c.ShouldBindJSON(&data); err != nil {
		logger.Error(err, "OpsCloudChangeFinishNotifyRequest:bind json error")
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	result, err := SubmitChangeFinishNotify(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}
