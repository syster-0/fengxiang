# Vendor 说明

本目录是 https://github.com/NanmiCoder/MediaCrawler 的**快照内嵌**（vendored copy），
上游锚点 commit：`60e66f2 docs: update README.md`（2026-09-16 浅克隆）。

- 为保证接收方 `git clone` 后开箱即跑，源码直接入库，不带上游 git 历史。
- 上游更新方式：备份本地改动（如有）→ 删除本目录 → `git clone --depth 1`
  上游仓库到本目录 → 复跑 `bash scripts/bootstrap.sh`（Windows 用 `bootstrap.ps1`）。
- 上游许可证：NON-COMMERCIAL LEARNING LICENSE 1.1（仅限非商业学习研究，
  禁止大规模爬取；商业用途必须停用该引擎、改接授权数据源或平台官方 API）。
