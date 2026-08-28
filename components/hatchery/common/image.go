package common

// CandidateImage 候选公有镜像信息
type CandidateImage struct {
	ImageId      string
	ImageName    string
	AgentType    string // 智能体类型，空值表示 openclaw（兼容存量）
	AgentVersion string // 智能体版本号
}

// CandidateImages 候选公有镜像列表，按优先级从高到低排列。
// SeedAvailableImages 会依次通过 DescribeImages 接口检查当前账号是否有权限使用，
// 将所有有权限的镜像导入数据库，并按此优先级选择第一个作为默认启用镜像。
// 对于存量场景（表中只有候选镜像），也会重新扫描并清理无权限的镜像。
var CandidateImages = []CandidateImage{
	// OpenClaw
	{ImageId: "img-idzg74s9", ImageName: "OpenClaw on Ubuntu 24.04", AgentType: "openclaw", AgentVersion: "2026.5.7"},
	{ImageId: "img-nmg7pw1r", ImageName: "OpenClaw on TencentOS Server 4", AgentType: "openclaw", AgentVersion: "2026.4.23"},
	{ImageId: "img-pf18atu9", ImageName: "OpenClaw on TencentOS Server 4 For Tencent", AgentType: "openclaw", AgentVersion: "2026.4.23"},
	// LightclawACE
	{ImageId: "img-0dvlda3b", ImageName: "LightClaw ACE on TencentOS Server 4", AgentType: "lightclawace", AgentVersion: "0.1.1"},
	// Hermes
	{ImageId: "img-al484uhr", ImageName: "Hermes Agent on Ubuntu 24.04", AgentType: "hermes", AgentVersion: "0.12.0"},
	{ImageId: "img-ppz9gfjn", ImageName: "Hermes Agent on TencentOS Server 4", AgentType: "hermes", AgentVersion: "0.12.0"},
	// DeepSeek TUI
	{ImageId: "img-khpujhmf", ImageName: "DeepSeek TUI on Ubuntu24.04", AgentType: "deepseektui", AgentVersion: "0.8.20"},
	// OpenCode
	{ImageId: "img-d8216ykb", ImageName: "OpenCode on Ubuntu24.04", AgentType: "opencode", AgentVersion: "1.14.41"},
}

// GetCandidateImage 获取指定官方候选公共镜像信息。
func GetCandidateImage(imageId string) (CandidateImage, bool) {
	for _, c := range CandidateImages {
		if c.ImageId == imageId {
			return c, true
		}
	}
	return CandidateImage{}, false
}

// IsCandidateImage 判断指定 imageId 是否属于候选公共镜像列表
func IsCandidateImage(imageId string) bool {
	_, ok := GetCandidateImage(imageId)
	return ok
}
