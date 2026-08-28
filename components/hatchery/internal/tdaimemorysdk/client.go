package tdaimemorysdk

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

// Client 是内部 TDAI Agent Memory SDK 客户端。
type Client struct {
	service string
	version string
	client  *common.Client
}

func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.SecretID) == "" {
		return nil, ErrEmptySecretID
	}
	if strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, ErrEmptySecretKey
	}

	if cfg.Service == "" {
		cfg.Service = DefaultService
	}
	if cfg.Version == "" {
		cfg.Version = DefaultVersion
	}
	if cfg.Region == "" {
		cfg.Region = DefaultRegion
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = cfg.Service + ".tencentcloudapi.com"
	}

	var cred common.CredentialIface
	if cfg.Token != "" {
		cred = common.NewTokenCredential(cfg.SecretID, cfg.SecretKey, cfg.Token)
	} else {
		cred = common.NewCredential(cfg.SecretID, cfg.SecretKey)
	}

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.ReqMethod = http.MethodPost
	cpf.HttpProfile.Endpoint = cfg.Endpoint
	cpf.HttpProfile.ReqTimeout = int((cfg.Timeout + time.Second - 1) / time.Second)

	cc := common.NewCommonClient(cred, cfg.Region, cpf)
	if cfg.RequestClient != "" {
		cc.WithRequestClient(cfg.RequestClient)
	}

	return &Client{
		service: cfg.Service,
		version: cfg.Version,
		client:  cc,
	}, nil
}

// Do 用于调用任意 Action（包括短期内部接口）。
// - action: 例如 DescribeInternalMemoryTask
// - req: 结构体/map，最终序列化为 JSON Body
// - resp: 期望解析到 Response 内层对象（不含最外层 Response 包裹）
// 返回 requestId，便于日志追踪。
func (c *Client) Do(ctx context.Context, action string, req any, resp any) (string, error) {
	if c == nil || c.client == nil {
		return "", hcommon.I18nError(i18n.MsgTDAISDKClientNotInit)
	}
	if strings.TrimSpace(action) == "" {
		return "", ErrEmptyAction
	}
	if ctx == nil {
		ctx = context.Background()
	}

	request := tchttp.NewCommonRequest(c.service, c.version, action)
	request.SetContext(ctx)

	payload := req
	if payload == nil {
		payload = map[string]any{}
	}

	// SetActionParameters 仅接受 map[string]any / string / []byte，
	// 若传入 struct 则先序列化为 JSON 再转为 map。
	switch payload.(type) {
	case map[string]any, string, []byte:
		// 直接使用
	default:
		b, err := json.Marshal(payload)
		if err != nil {
			return "", hcommon.I18nRichError(err, i18n.MsgProviderMarshalRequest)
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return "", hcommon.I18nRichError(err, i18n.MsgTDAISDKConvertToMapFailed)
		}
		payload = m
	}

	if err := request.SetActionParameters(payload); err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgSetActionParamsFailed)
	}

	rawResp := tchttp.NewCommonResponse()
	if err := c.client.Send(request, rawResp); err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgTDAISDKSendRequestFailed)
	}

	return decodeActionResponse(rawResp.GetBody(), resp)
}

// DescribeAgentInstance 是一个简单读接口示例（复制现有 tdai sdk 读接口）。
func (c *Client) DescribeAgentInstance(ctx context.Context, req *DescribeAgentInstanceRequest) (*DescribeAgentInstanceResponse, error) {
	if req == nil {
		req = &DescribeAgentInstanceRequest{}
	}
	out := &DescribeAgentInstanceResponse{}
	if _, err := c.Do(ctx, "DescribeAgentInstance", req, out); err != nil {
		return nil, err
	}
	return out, nil
}

type responseEnvelope struct {
	Response json.RawMessage `json:"Response"`
}

type responseMeta struct {
	Error     *responseErr `json:"Error,omitempty"`
	RequestID string       `json:"RequestId,omitempty"`
}

type responseErr struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

func decodeActionResponse(body []byte, out any) (string, error) {
	if len(body) == 0 {
		return "", hcommon.I18nError(i18n.MsgTDAISDKEmptyResponseBody)
	}

	var env responseEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgTDAISDKDecodeEnvelopeFailed)
	}
	if len(env.Response) == 0 {
		return "", hcommon.I18nError(i18n.MsgTDAISDKMissingResponseField)
	}

	var meta responseMeta
	if err := json.Unmarshal(env.Response, &meta); err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgTDAISDKDecodeMetaFailed)
	}
	if meta.Error != nil {
		return meta.RequestID, &APIError{
			Code:      meta.Error.Code,
			Message:   meta.Error.Message,
			RequestID: meta.RequestID,
		}
	}

	if out != nil {
		if err := json.Unmarshal(env.Response, out); err != nil {
			return meta.RequestID, hcommon.I18nRichError(err, i18n.MsgTDAISDKDecodePayloadFailed)
		}
	}

	return meta.RequestID, nil
}

// ========== Memory Pro（VDB 实例）API ==========

// CreateMemoryProInstance 开通 Memory Pro 服务（创建 VDB 实例）。
func (c *Client) CreateMemoryProInstance(ctx context.Context, req *CreateMemoryProInstanceRequest) (*CreateMemoryProInstanceResponse, error) {
	out := &CreateMemoryProInstanceResponse{}
	if _, err := c.Do(ctx, "CreateMemoryProInstance", req, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DescribeMemoryProInstances 查询 Memory Pro 实例详情。
func (c *Client) DescribeMemoryProInstances(ctx context.Context, req *DescribeMemoryProInstancesRequest) (*DescribeMemoryProInstancesResponse, error) {
	if req == nil {
		req = &DescribeMemoryProInstancesRequest{}
	}
	out := &DescribeMemoryProInstancesResponse{}
	if _, err := c.Do(ctx, "DescribeMemoryProInstances", req, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ModifyMemoryProInstance 修改 Memory Pro 实例参数（如扩容 MemoryLimit）。
func (c *Client) ModifyMemoryProInstance(ctx context.Context, req *ModifyMemoryProInstanceRequest) (*ModifyMemoryProInstanceResponse, error) {
	out := &ModifyMemoryProInstanceResponse{}
	if _, err := c.Do(ctx, "ModifyMemoryProInstance", req, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteMemoryProInstance 关闭 Memory Pro 服务（释放 VDB 实例）。
func (c *Client) DeleteMemoryProInstance(ctx context.Context, req *DeleteMemoryProInstanceRequest) (*DeleteMemoryProInstanceResponse, error) {
	out := &DeleteMemoryProInstanceResponse{}
	if _, err := c.Do(ctx, "DeleteMemoryProInstance", req, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ========== MemSpace（VDB Database / 记忆库）API ==========

// CreateMemSpace 创建记忆库（VDB database）。
func (c *Client) CreateMemSpace(ctx context.Context, req *CreateMemSpaceRequest) (*CreateMemSpaceResponse, error) {
	out := &CreateMemSpaceResponse{}
	if _, err := c.Do(ctx, "CreateMemSpace", req, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DescribeMemSpaces 查询记忆库列表。
func (c *Client) DescribeMemSpaces(ctx context.Context, req *DescribeMemSpacesRequest) (*DescribeMemSpacesResponse, error) {
	if req == nil {
		req = &DescribeMemSpacesRequest{}
	}
	out := &DescribeMemSpacesResponse{}
	if _, err := c.Do(ctx, "DescribeMemSpaces", req, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DescribeMemSpaceRecord 查询记忆库数据记录。
func (c *Client) DescribeMemSpaceRecord(ctx context.Context, req *DescribeMemSpaceRecordRequest) (*DescribeMemSpaceRecordResponse, error) {
	out := &DescribeMemSpaceRecordResponse{}
	if _, err := c.Do(ctx, "DescribeMemSpaceRecord", req, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteMemSpace 删除记忆库（VDB database）。
func (c *Client) DeleteMemSpace(ctx context.Context, req *DeleteMemSpaceRequest) (*DeleteMemSpaceResponse, error) {
	out := &DeleteMemSpaceResponse{}
	if _, err := c.Do(ctx, "DeleteMemSpace", req, out); err != nil {
		return nil, err
	}
	return out, nil
}
