package controller

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
)

// UploadArchiveToSMH 将 CVM 上的备份压缩包分块上传到 SMH。
// 复用 openclaw_upgrade.go 中的上传流程，提取为公共函数供 upgrade 和 doctor snapshot 共用。
// 返回 SMH fileKey。
func UploadArchiveToSMH(
	ctx context.Context,
	instanceId string,
	runtimeUser string,
	archivePath string,
	archiveSize int64,
) (string, error) {
	cred, err := PrepareSMHCommonUpload(
		ctx, instanceId, archivePath, archiveSize)
	if err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgSMHUploadCredFailed)
	}
	return uploadArchiveCore(
		ctx, instanceId, runtimeUser, archivePath, cred, "")
}

// UploadArchiveToSMHWithKey 与 UploadArchiveToSMH 功能相同，
// 但允许指定自定义的 SMH fileKey（而非由 instanceId + filename 自动生成）。
func UploadArchiveToSMHWithKey(
	ctx context.Context,
	instanceId string,
	runtimeUser string,
	archivePath string,
	archiveSize int64,
	fileKey string,
) (string, error) {
	cred, err := PrepareMigrationUpload(ctx, fileKey, archiveSize)
	if err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgSMHUploadCredFailed)
	}
	return uploadArchiveCore(
		ctx, instanceId, runtimeUser, archivePath, cred, fileKey)
}

// uploadArchiveCore 是分块上传 + 确认的公共流程。
// customFileKey 仅用于日志（""表示由 cred.FileKey 自动生成）。
func uploadArchiveCore(
	ctx context.Context,
	instanceId string,
	runtimeUser string,
	archivePath string,
	cred *SMHUploadCredential,
	customFileKey string,
) (string, error) {
	log := Logger(ctx).With("instance_id", instanceId)

	if cred.PartURLTemplate == "" {
		return "", hcommon.I18nError(i18n.MsgUpgradeSMHCredMissingChunkURL)
	}

	if customFileKey != "" {
		log.Info("[SMHUpload] 开始分块上传（自定义 key）",
			"fileKey", customFileKey,
			"totalParts", cred.TotalParts,
			"partSize", cred.PartSize)
	} else {
		log.Info("[SMHUpload] 开始分块上传",
			"totalParts", cred.TotalParts,
			"partSize", cred.PartSize)
	}

	uploadScriptTemplate, err := LoadScript("upload_to_smh.sh")
	if err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgUpgradeLoadUploadScriptFailed)
	}

	for partNum := 1; partNum <= cred.TotalParts; partNum++ {
		if err := uploadOnePart(
			ctx, log, instanceId, runtimeUser, archivePath,
			cred, uploadScriptTemplate, partNum,
		); err != nil {
			return "", err
		}
	}

	log.Info("[SMHUpload] 所有分块上传完成，确认上传")
	if err := ConfirmSMHCommonUpload(ctx, cred.ConfirmKey); err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgUpgradeConfirmSMHUploadFailed)
	}

	log.Info("[SMHUpload] 上传成功",
		"fileKey", cred.FileKey)
	return cred.FileKey, nil
}

// uploadOnePart 渲染并执行单个分块的上传脚本。
func uploadOnePart(
	ctx context.Context,
	log *slog.Logger,
	instanceId string,
	runtimeUser string,
	archivePath string,
	cred *SMHUploadCredential,
	scriptTemplate string,
	partNum int,
) error {
	partURL := strings.ReplaceAll(
		cred.PartURLTemplate,
		"{partNumber}",
		strconv.Itoa(partNum))
	offset := int64(partNum-1) * cred.PartSize
	uploadURLB64 := base64.StdEncoding.EncodeToString(
		[]byte(partURL))

	headerCount := len(cred.PartHeaders)
	headerKVLines := ""
	if headerCount > 0 {
		i := 0
		for k, v := range cred.PartHeaders {
			headerKVLines += fmt.Sprintf(
				"HEADER_%d_KEY=%q\n", i, k)
			headerKVLines += fmt.Sprintf(
				"HEADER_%d_VAL_B64=%q\n", i,
				base64.StdEncoding.EncodeToString(
					[]byte(v)))
			i++
		}
	}

	uploadScript := scriptTemplate
	uploadScript = strings.ReplaceAll(uploadScript,
		`ARCHIVE_PATH="{{archivepath}}"`,
		fmt.Sprintf(`ARCHIVE_PATH="%s"`, archivePath))
	uploadScript = strings.ReplaceAll(uploadScript,
		`UPLOAD_URL_B64="{{uploadurlb64}}"`,
		fmt.Sprintf(`UPLOAD_URL_B64="%s"`, uploadURLB64))
	uploadScript = strings.ReplaceAll(uploadScript,
		`OFFSET="{{offset}}"`,
		fmt.Sprintf(`OFFSET="%s"`,
			strconv.FormatInt(offset, 10)))
	uploadScript = strings.ReplaceAll(uploadScript,
		`PART_SIZE="{{partsize}}"`,
		fmt.Sprintf(`PART_SIZE="%s"`,
			strconv.FormatInt(cred.PartSize, 10)))
	uploadScript = strings.ReplaceAll(uploadScript,
		`PART_NUMBER="{{partnumber}}"`,
		fmt.Sprintf(`PART_NUMBER="%s"`,
			strconv.Itoa(partNum)))
	uploadScript = strings.ReplaceAll(uploadScript,
		`TOTAL_PARTS="{{totalparts}}"`,
		fmt.Sprintf(`TOTAL_PARTS="%s"`,
			strconv.Itoa(cred.TotalParts)))
	uploadScript = strings.ReplaceAll(uploadScript,
		`HEADER_COUNT={{headercount}}`,
		fmt.Sprintf(`HEADER_COUNT=%d`, headerCount))
	uploadScript = strings.ReplaceAll(uploadScript,
		`{{headerkvlines}}`, headerKVLines)

	tmpName := fmt.Sprintf(
		"_upload_smh_%s_part%d_%d.sh",
		instanceId, partNum, time.Now().UnixNano())
	RegisterInlineScript(tmpName, uploadScript)
	defer UnregisterInlineScript(tmpName)

	log.Info("[SMHUpload] 上传分块",
		"partNumber", partNum,
		"totalParts", cred.TotalParts)

	_, runErr := RunScript(
		ctx, instanceId, tmpName, 300,
		runtimeUser, nil, nil)

	if runErr != nil {
		log.Warn("[SMHUpload] 分块上传失败，SMH multipart 会话未确认，服务端将自动清理",
			"confirm_key", cred.ConfirmKey,
			"part", partNum)
		return hcommon.I18nRichError(runErr, i18n.MsgSMHUploadPartFailed, partNum, cred.TotalParts)
	}
	return nil
}
