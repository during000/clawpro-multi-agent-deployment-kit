package controller

// OffloadBackendURLs 地域→Offload 后端地址映射表。
// 不同地域有专属的后端请求地址，各地域域名、端口均不相同。
// 后续新增地域时，只需在此处追加即可。
// ⚠️ 当前所有地域统一指向测试环境地址，上线前务必改回各地域专属地址！
// 正式地址：ap-chengdu → cd.test.polaris:32200, ap-guangzhou → gz.test.polaris:12353, ap-nanjing → nj.test.polaris:15425
var OffloadBackendURLs = map[string]string{
	"ap-beijing":   "https://memory.tdai.tencentyun.com",
	"ap-guangzhou": "https://memory.tdai.tencentyun.com",
	"ap-nanjing":   "https://memory.tdai.tencentyun.com",
	"ap-shanghai":  "https://memory.tdai.tencentyun.com",
}

// GetOffloadBackendURL 根据当前地域（CVMRegion）返回对应的 Offload 后端地址。
// 若当前地域没有配置 Offload 后端，返回空字符串（调用方应判断并跳过 offload）。
func GetOffloadBackendURL() string {
	return OffloadBackendURLs[CVMRegion]
}
