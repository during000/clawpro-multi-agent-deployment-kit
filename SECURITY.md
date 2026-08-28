# 安全说明

- 本仓库保持私有，只向明确添加的 GitHub 协作者开放。
- Release 包不包含现有测试环境的域名、IP、Cookie、个人 Token、CloudAgent 密钥或 iMate 凭证。
- 服务器首次安装时生成独立管理员密码、后台 Token 和 Cookie Secret，保存在服务器权限为 `0600` 的配置文件中。
- 用户电脑应使用自己的 ClawPro 用户 API Token，不得使用服务器后台管理员 Token。
- 必须通过 HTTPS/WSS 对外提供服务；HTTP 80 仅作为服务器内部或上游 TLS 网关的源站。
- 这是测试/演示部署套件，不承诺生产级高可用、容灾和合规能力。
