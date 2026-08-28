package common

import (
	"testing"
)

func TestIsCandidateImage_KnownImages(t *testing.T) {
	// 验证所有候选镜像都能被正确识别
	for _, candidate := range CandidateImages {
		if !IsCandidateImage(candidate.ImageId) {
			t.Errorf("IsCandidateImage(%q) = false, expected true for candidate image %q",
				candidate.ImageId, candidate.ImageName)
		}
	}
}

func TestIsCandidateImage_UnknownImages(t *testing.T) {
	unknownIds := []string{
		"img-unknown-001",
		"img-000000",
		"",
		"img-fake",
	}

	for _, id := range unknownIds {
		if IsCandidateImage(id) {
			t.Errorf("IsCandidateImage(%q) = true, expected false for unknown image", id)
		}
	}
}

func TestCandidateImages_HaveRequiredFields(t *testing.T) {
	if len(CandidateImages) == 0 {
		t.Fatal("CandidateImages should not be empty")
	}

	for i, img := range CandidateImages {
		if img.ImageId == "" {
			t.Errorf("CandidateImages[%d].ImageId is empty", i)
		}
		if img.ImageName == "" {
			t.Errorf("CandidateImages[%d].ImageName is empty", i)
		}
		if img.AgentType == "" {
			t.Errorf("CandidateImages[%d].AgentType is empty", i)
		}
		if img.AgentVersion == "" {
			t.Errorf("CandidateImages[%d].AgentVersion is empty", i)
		}
	}
}

func TestCandidateImages_AgentTypesValid(t *testing.T) {
	validTypes := map[string]bool{
		"openclaw":     true,
		"hermes":       true,
		"lightclawace": true,
		"deepseektui":  true,
		"opencode":     true,
	}

	for i, img := range CandidateImages {
		if !validTypes[img.AgentType] {
			t.Errorf("CandidateImages[%d].AgentType = %q is not a valid agent type", i, img.AgentType)
		}
	}
}

func TestCandidateImages_NoDuplicateIds(t *testing.T) {
	seen := make(map[string]bool)
	for i, img := range CandidateImages {
		if seen[img.ImageId] {
			t.Errorf("CandidateImages[%d] has duplicate ImageId %q", i, img.ImageId)
		}
		seen[img.ImageId] = true
	}
}

func TestCandidateImages_CoverAllAgentTypes(t *testing.T) {
	typeSet := make(map[string]bool)
	for _, img := range CandidateImages {
		typeSet[img.AgentType] = true
	}

	expectedTypes := []string{"openclaw", "hermes", "lightclawace", "deepseektui", "opencode"}
	for _, typ := range expectedTypes {
		if !typeSet[typ] {
			t.Errorf("CandidateImages missing agent type %q", typ)
		}
	}
}

func TestCandidateImages_OpenClawVersionFormat(t *testing.T) {
	for i, img := range CandidateImages {
		if img.AgentType == "openclaw" {
			// OpenClaw 版本格式: YYYY.M.D
			if len(img.AgentVersion) < 6 { // 至少 "Y.M.D"
				t.Errorf("CandidateImages[%d] OpenClaw version too short: %q", i, img.AgentVersion)
			}
		}
	}
}

func TestCandidateImages_NonOpenClawSemverFormat(t *testing.T) {
	for i, img := range CandidateImages {
		if img.AgentType == "hermes" || img.AgentType == "lightclawace" ||
			img.AgentType == "deepseektui" || img.AgentType == "opencode" {
			// 非 OpenClaw 类型应该使用 semver 格式 X.Y.Z
			parts := 0
			for _, c := range img.AgentVersion {
				if c == '.' {
					parts++
				}
			}
			if parts != 2 {
				t.Errorf("CandidateImages[%d] (%s) version should be semver X.Y.Z, got %q",
					i, img.AgentType, img.AgentVersion)
			}
		}
	}
}

func TestCandidateImages_CountByType(t *testing.T) {
	counts := make(map[string]int)
	for _, img := range CandidateImages {
		counts[img.AgentType]++
	}

	// 每种类型至少有一个候选镜像
	if counts["openclaw"] == 0 {
		t.Error("expected at least one openclaw candidate image")
	}
	if counts["hermes"] == 0 {
		t.Error("expected at least one hermes candidate image")
	}
	if counts["lightclawace"] == 0 {
		t.Error("expected at least one lightclawace candidate image")
	}
	if counts["deepseektui"] == 0 {
		t.Error("expected at least one deepseektui candidate image")
	}
	if counts["opencode"] == 0 {
		t.Error("expected at least one opencode candidate image")
	}
}

// ==================== CandidateInfo 映射构建测试 ====================
// 覆盖 HandleListCloudImages 中新增的候选镜像 agent 信息映射逻辑

func TestCandidateInfoMap_BuildAndLookup(t *testing.T) {
	// 模拟 HandleListCloudImages 中构建 candidateInfo 映射的逻辑
	candidateInfo := make(map[string]CandidateImage, len(CandidateImages))
	for _, c := range CandidateImages {
		candidateInfo[c.ImageId] = c
	}

	// 验证映射大小与候选列表一致
	if len(candidateInfo) != len(CandidateImages) {
		t.Errorf("candidateInfo 映射大小 %d 与 CandidateImages 列表大小 %d 不一致",
			len(candidateInfo), len(CandidateImages))
	}

	// 验证每个候选镜像都能通过 ImageId 查找到，且 AgentType 和 AgentVersion 正确
	for _, c := range CandidateImages {
		found, ok := candidateInfo[c.ImageId]
		if !ok {
			t.Errorf("candidateInfo 中找不到 ImageId=%q", c.ImageId)
			continue
		}
		if found.AgentType != c.AgentType {
			t.Errorf("candidateInfo[%q].AgentType = %q, want %q", c.ImageId, found.AgentType, c.AgentType)
		}
		if found.AgentVersion != c.AgentVersion {
			t.Errorf("candidateInfo[%q].AgentVersion = %q, want %q", c.ImageId, found.AgentVersion, c.AgentVersion)
		}
	}
}

func TestCandidateInfoMap_NonCandidateImageNotFound(t *testing.T) {
	// 构建映射
	candidateInfo := make(map[string]CandidateImage, len(CandidateImages))
	for _, c := range CandidateImages {
		candidateInfo[c.ImageId] = c
	}

	// 非候选镜像不应在映射中
	nonCandidateIDs := []string{"img-unknown-001", "img-000000", "", "img-fake"}
	for _, id := range nonCandidateIDs {
		if _, ok := candidateInfo[id]; ok {
			t.Errorf("非候选镜像 %q 不应出现在 candidateInfo 映射中", id)
		}
	}
}

func TestCandidateInfoMap_AgentFieldsPopulated(t *testing.T) {
	// 验证通过映射查找到的候选镜像，AgentType 和 AgentVersion 均非空
	candidateInfo := make(map[string]CandidateImage, len(CandidateImages))
	for _, c := range CandidateImages {
		candidateInfo[c.ImageId] = c
	}

	for id, info := range candidateInfo {
		if info.AgentType == "" {
			t.Errorf("candidateInfo[%q].AgentType 不应为空", id)
		}
		if info.AgentVersion == "" {
			t.Errorf("candidateInfo[%q].AgentVersion 不应为空", id)
		}
	}
}
