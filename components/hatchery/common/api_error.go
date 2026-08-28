package common

import (
	"context"
	"errors"
	"fmt"

	"hatchery/i18n"

	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
)

// RichError 携带额外上下文（如 RequestId、InstanceId）的错误类型。
type RichError struct {
	Detail       string         // 详细信息（如脚本输出）
	RequestId    string         // 腾讯云 RequestId，空表示非 SDK 错误
	BizRequestId string         // 业务请求 ID，空表示未关联业务请求
	InstanceId   string         // 操作的实例 ID，空表示无关联实例
	CustomData   map[string]any // 自定义数据，空表示无自定义数据

	i18nPrefix  []any
	i18nMessage *i18n.KeyAndArgs
	i18nDetail  *i18n.KeyAndArgs

	cause []error
}

// i18nError 创建纯消息错误（无原始 cause）
func I18nError(key i18n.Key, args ...any) *RichError {
	return &RichError{
		i18nMessage: &i18n.KeyAndArgs{Key: key, Args: args},
	}
}

// i18nRichError 创建带原始错误的消息错误。
// 自动从 err 中提取 Detail / RequestId，err 放入 cause
func I18nRichError(err error, key i18n.Key, args ...any) *RichError {
	re := &RichError{
		i18nMessage: &i18n.KeyAndArgs{Key: key, Args: args},
		cause:       make([]error, 0),
	}

	if err != nil {
		var sdkErr *sdkerrors.TencentCloudSDKError
		if errors.As(err, &sdkErr) {
			if sdkErr == nil {
				return re
			}
			re.Detail = sdkErr.GetMessage()
			re.RequestId = sdkErr.GetRequestId()
			re.cause = append(re.cause, sdkErr)
			return re
		}

		var ire *RichError
		if errors.As(err, &ire) {
			if ire == nil {
				return re
			}
			re.Detail = ire.Detail
			re.RequestId = ire.RequestId
			re.BizRequestId = ire.BizRequestId
			re.InstanceId = ire.InstanceId
			re.CustomData = ire.CustomData
			re.i18nDetail = ire.i18nDetail
			re.i18nPrefix = ire.i18nPrefix

			re.cause = append(re.cause, ire)
			return re
		}

		re.cause = append(re.cause, err)
	}

	return re
}

func (e *RichError) Error() string {
	if e.RequestId != "" {
		return fmt.Sprintf("%s (RequestId: %s)", e.ErrorMessage(context.Background()), e.RequestId)
	}
	return e.ErrorMessage(context.Background())
}

func (re *RichError) ErrorMessageWithCauses(ctx context.Context) string {
	if len(re.cause) == 0 {
		return ""
	}
	return fmt.Sprintf("%s: %v", re.ErrorMessage(ctx), re.cause)
}

func (re *RichError) ErrorMessage(ctx context.Context) string {
	if re.i18nMessage != nil {
		prefix := ""
		if re.i18nPrefix != nil {
			for _, i18nPrefix := range re.i18nPrefix {
				if p, ok := i18nPrefix.(string); ok {
					prefix += p
				} else if kv, ok := i18nPrefix.(i18n.KeyAndArgs); ok {
					prefix += i18n.T(ctx, kv.Key, kv.Args...)
				}
				prefix += ": "
			}
		}
		msg := i18n.T(ctx, re.i18nMessage.Key, re.i18nMessage.Args...)

		return prefix + msg
	}
	return ""
}

func ErrorMessageWithCausesCtx(ctx context.Context, err error) string {
	if err == nil {
		return ""
	}
	var re *RichError
	if errors.As(err, &re) {
		if re == nil {
			return ""
		}
		return re.ErrorMessageWithCauses(ctx)
	}
	return err.Error()
}

// errorDetailCtx 从 RichError 中提取 Message，如果存在 i18n Message 则进行翻译
// 如果不是 RichError 类型的错误则返回 err.Error()
func ErrorMessageWithCtx(ctx context.Context, err error) string {
	if err == nil {
		return ""
	}
	var re *RichError
	if errors.As(err, &re) {
		// 如果传入的是一个 nil 的 RichError 类型变量
		// err 并不为 nil，但是 re 为 nil
		if re == nil {
			return ""
		}
		return re.ErrorMessage(ctx)
	}
	return err.Error()
}

// errorDetailCtx 从 RichError 中提取 Detail，如果存在 i18n Detail 则进行翻译
// 如果不是 RichError 类型的错误则返回空字符串
func ErrorDetailWithCtx(ctx context.Context, err error) string {
	var re *RichError
	if errors.As(err, &re) {
		if re.i18nDetail != nil {
			return i18n.T(ctx, re.i18nDetail.Key, re.i18nDetail.Args...)
		}
		return re.Detail
	}
	return ""
}

// errorRequestId 从 error 提取腾讯云 RequestId，非 RichError 返回空。
func ErrorRequestId(err error) string {
	var re *RichError
	if errors.As(err, &re) {
		return re.RequestId
	}
	return ""
}

// errorBizRequestId 从 error 提取业务请求 ID，非 RichError 返回空。
func ErrorBizRequestId(err error) string {
	var re *RichError
	if errors.As(err, &re) {
		return re.BizRequestId
	}
	return ""
}

// errorInstanceId 从 error 提取实例 ID，非 RichError 返回空。
func ErrorInstanceId(err error) string {
	var re *RichError
	if errors.As(err, &re) {
		return re.InstanceId
	}
	return ""
}

func EnsureRichErrorOrPanic(err error) *RichError {
	if err == nil {
		return nil
	}

	var re *RichError
	if errors.As(err, &re) {
		return re
	} else {
		panic(fmt.Sprintf("writeError 只接收 RichError，但是传入的 error 为 %#v", err))
	}
}

func (r *RichError) DeepCopy() *RichError {
	if r == nil {
		return nil
	}

	cp := &RichError{
		Detail:       r.Detail,
		RequestId:    r.RequestId,
		BizRequestId: r.BizRequestId,
		InstanceId:   r.InstanceId,
	}

	// 深拷贝 CustomData map
	// 如果 value 实现 DeepCopy() any 接口，则调用 DeepCopy 方法进行深拷贝，否则直接复制
	if r.CustomData != nil {
		cp.CustomData = make(map[string]any, len(r.CustomData))
		for k, v := range r.CustomData {
			if vv, ok := v.(interface{ DeepCopy() any }); ok {
				cp.CustomData[k] = vv.DeepCopy()
			} else {
				cp.CustomData[k] = v
			}
		}
	}

	// 深拷贝 i18nPrefix slice
	if r.i18nPrefix != nil {
		cp.i18nPrefix = make([]any, len(r.i18nPrefix))
		for i, prefix := range r.i18nPrefix {
			switch p := prefix.(type) {
			case string:
				cp.i18nPrefix[i] = p
			case i18n.KeyAndArgs:
				ka := i18n.KeyAndArgs{
					Key:  p.Key,
					Args: make([]any, len(p.Args)),
				}
				copy(ka.Args, p.Args)
				cp.i18nPrefix[i] = ka
			default:
				cp.i18nPrefix[i] = p
			}
		}
	}

	// 深拷贝 i18nMessage
	if r.i18nMessage != nil {
		cp.i18nMessage = &i18n.KeyAndArgs{
			Key:  r.i18nMessage.Key,
			Args: make([]any, len(r.i18nMessage.Args)),
		}
		copy(cp.i18nMessage.Args, r.i18nMessage.Args)
	}

	// 深拷贝 i18nDetail
	if r.i18nDetail != nil {
		cp.i18nDetail = &i18n.KeyAndArgs{
			Key:  r.i18nDetail.Key,
			Args: make([]any, len(r.i18nDetail.Args)),
		}
		copy(cp.i18nDetail.Args, r.i18nDetail.Args)
	}

	// 深拷贝 cause slice: RichError 递归深拷贝，其他 error 共用指针
	if r.cause != nil {
		cp.cause = make([]error, len(r.cause))
		for i, err := range r.cause {
			if re, ok := err.(*RichError); ok {
				cp.cause[i] = re.DeepCopy()
			} else {
				cp.cause[i] = err
			}
		}
	}

	return cp
}

// 增加一个不进行翻译的前缀
func (r *RichError) WithPrefix(prefix string) *RichError {
	rCopy := r.DeepCopy()

	if rCopy.i18nPrefix != nil {
		newPrefixes := make([]any, len(rCopy.i18nPrefix)+1)
		newPrefixes[0] = prefix
		copy(newPrefixes[1:], rCopy.i18nPrefix)
		rCopy.i18nPrefix = newPrefixes
	} else {
		rCopy.i18nPrefix = []any{prefix}
	}

	return rCopy
}

func (r *RichError) WithDetail(detail string) *RichError {
	rCopy := r.DeepCopy()
	rCopy.Detail = detail
	return rCopy
}

func (r *RichError) WithI18nPrefix(key i18n.Key, args ...any) *RichError {
	rCopy := r.DeepCopy()

	prefix := i18n.KeyAndArgs{Key: key, Args: args}
	if rCopy.i18nPrefix != nil {
		newPrefixes := make([]any, len(rCopy.i18nPrefix)+1)
		newPrefixes[0] = prefix
		copy(newPrefixes[1:], rCopy.i18nPrefix)
		rCopy.i18nPrefix = newPrefixes
	} else {
		rCopy.i18nPrefix = []any{prefix}
	}
	return rCopy
}

func (r *RichError) WithI18nDetail(key i18n.Key, args ...any) *RichError {
	rCopy := r.DeepCopy()
	rCopy.i18nDetail = &i18n.KeyAndArgs{Key: key, Args: args}
	return rCopy
}

func (r *RichError) WithCustomData(customData map[string]interface{}) *RichError {
	rCopy := r.DeepCopy()
	rCopy.CustomData = customData
	return rCopy
}

func (r *RichError) WithBizRequestId(bizRequestId string) *RichError {
	rCopy := r.DeepCopy()
	rCopy.BizRequestId = bizRequestId
	return rCopy
}

func (r *RichError) WithInstanceId(instanceId string) *RichError {
	rCopy := r.DeepCopy()
	rCopy.InstanceId = instanceId
	return rCopy
}

func (r *RichError) WithRequestId(requestId string) *RichError {
	rCopy := r.DeepCopy()
	rCopy.RequestId = requestId
	return rCopy
}

func (r *RichError) WithCause(err error) *RichError {
	if err == nil {
		return r
	}

	// 如果传入 RichError 类型的 nil，则 err 不为 nil
	// 需要将类型转换为 RichError 再检查是否为 nil
	var re *RichError
	if errors.As(err, &re) && re == nil {
		return r
	}

	rCopy := r.DeepCopy()

	if rCopy.cause == nil {
		rCopy.cause = []error{err}
	} else {
		rCopy.cause = append(rCopy.cause, err)
	}
	return rCopy
}

// ======================================================
// 用来兼容部分需要进行 i18n 又需要通过 errors.Is 检测原始 error 的逻辑
// ======================================================
func (r *RichError) Unwrap() []error {
	return r.cause
}

// Is 支持 errors.Is 匹配：当两个 RichError 的 i18nMessage.Key 相等时视为同一错误。
// 这使得 sentinel RichError 变量可以通过 errors.Is 检测到。
// 如果该函数返回 false，errors.Is 会继续使用 Unwrap 进行比较
func (r *RichError) Is(target error) bool {
	t, ok := target.(*RichError)
	if !ok {
		return false
	}
	if r.i18nMessage == nil || t.i18nMessage == nil {
		return false
	}
	return r.i18nMessage.Key.String() == t.i18nMessage.Key.String()
}
