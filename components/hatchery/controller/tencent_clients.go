package controller

import (
	"context"

	cbs "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs/v20170312"
	cls "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cls/v20201016"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	csip "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/csip/v20221121"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	sts "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sts/v20180813"
	tag "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tag/v20180813"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// 腾讯云 SDK Client 工厂函数
// ===================================
// 每次调用返回一个新的 client 实例，凭证通过 getCredential(ctx) 统一获取。
// 后续如有性能需求可再引入池化/缓存优化。

// GetCVMClient 返回一个新的 cvm.Client。
func GetCVMClient(ctx context.Context) (*cvm.Client, error) {
	cred, err := getCredential(ctx)
	if err != nil {
		return nil, err
	}
	cpf := profile.NewClientProfile()
	return cvm.NewClient(cred, CVMRegion, cpf)
}

// GetCBSClient 返回一个新的 cbs.Client。
func GetCBSClient(ctx context.Context) (*cbs.Client, error) {
	cred, err := getCredential(ctx)
	if err != nil {
		return nil, err
	}
	cpf := profile.NewClientProfile()
	return cbs.NewClient(cred, CVMRegion, cpf)
}

// GetVPCClient 返回一个新的 vpc.Client。
func GetVPCClient(ctx context.Context) (*vpc.Client, error) {
	cred, err := getCredential(ctx)
	if err != nil {
		return nil, err
	}
	cpf := profile.NewClientProfile()
	return vpc.NewClient(cred, CVMRegion, cpf)
}

// GetSTSClient 返回一个新的 sts.Client。
func GetSTSClient(ctx context.Context) (*sts.Client, error) {
	cred, err := getCredential(ctx)
	if err != nil {
		return nil, err
	}
	cpf := profile.NewClientProfile()
	return sts.NewClient(cred, CVMRegion, cpf)
}

// GetCLSClient 返回一个新的 cls.Client。
func GetCLSClient(ctx context.Context) (*cls.Client, error) {
	cred, err := getCredential(ctx)
	if err != nil {
		return nil, err
	}
	cpf := profile.NewClientProfile()
	return cls.NewClient(cred, CVMRegion, cpf)
}

// GetTagClient 返回一个新的 tag.Client（标签服务没有 region 概念，传空字符串）。
func GetTagClient(ctx context.Context) (*tag.Client, error) {
	cred, err := getCredential(ctx)
	if err != nil {
		return nil, err
	}
	cpf := profile.NewClientProfile()
	return tag.NewClient(cred, "", cpf)
}

// GetCSIPClient 返回一个新的 csip.Client（Cloud Security Intelligence Platform）。
// 用于 Skill 安全扫描、SkillScan 相关 API
func GetCSIPClient(ctx context.Context) (*csip.Client, error) {
	cred, err := getCredential(ctx)
	if err != nil {
		return nil, err
	}
	cpf := profile.NewClientProfile()
	return csip.NewClient(cred, CVMRegion, cpf)
}
