package main

// repo 子命令：分发自感知。
//
// 专家包以 git 仓库形态对外分享（包根即仓库根）。接收方 clone 后，
// 发送方持续向远端推送知识库增量（wiki/、sources/、bin/ 等）；
// 接收方通过 `tvop repo status` 发现落后、`tvop repo pull` 安全拉取。
// doctor 也会报告同步状态，使专家在会话里"自己意识到需要拉取"。

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func cmdRepo(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, "用法：tvop repo <status|pull>\n")
		return exitInvalid
	}
	w, err := openWiki()
	if err != nil {
		return fail(err)
	}
	repo := w.RepoRoot()
	if _, statErr := os.Stat(filepath.Join(repo, ".git")); statErr != nil {
		fmt.Fprintln(os.Stderr, "[invalid] repo: 当前目录不是 git 仓库（zip 直解的包没有更新通道；请改用 git clone 分发）")
		return exitInvalid
	}
	switch args[0] {
	case "status":
		return repoStatus(repo)
	case "pull":
		return repoPull(repo)
	default:
		fmt.Fprintf(os.Stderr, "未知子命令：repo %s（可选 status | pull）\n", args[0])
		return exitInvalid
	}
}

// git 运行封装：15s 超时（fetch/pull 是网络动作，不允许挂死 CLI）。
func gitRun(repo string, timeout time.Duration, gitArgs ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", gitArgs...)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func repoStatus(repo string) int {
	fmt.Println("tvop repo status")
	fmt.Println("  仓库根 :", repo)

	branch, err := gitRun(repo, 5*time.Second, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		fmt.Println("  分支   : 读取失败（", err, "）")
		return exitInvalid
	}
	fmt.Println("  分支   :", branch)

	remote, rerr := gitRun(repo, 5*time.Second, "remote", "get-url", "origin")
	if rerr != nil || remote == "" {
		fmt.Println("  远端   : 未配置 —— 无法感知上游更新")
		fmt.Println("           发送方：git remote add origin <url> && git push -u origin master")
		return exitOK
	}
	fmt.Println("  远端   :", remote)

	head, _ := gitRun(repo, 5*time.Second, "log", "-1", "--format=%h %ad %s", "--date=short")
	fmt.Println("  本地   :", head)

	fmt.Print("  联网比对: ")
	if _, ferr := gitRun(repo, 20*time.Second, "fetch", "--quiet", "origin"); ferr != nil {
		fmt.Println("失败（离线或远端不可达），以本地为准")
		return exitOK
	}
	behind, _ := gitRun(repo, 5*time.Second,
		"rev-list", "--count", "HEAD..origin/"+branch)
	ahead, _ := gitRun(repo, 5*time.Second,
		"rev-list", "--count", "origin/"+branch+"..HEAD")
	fmt.Printf("落后 %s 个提交 / 领先 %s 个提交\n", behind, ahead)
	if behind != "0" {
		fmt.Printf("  ⚠ 本地落后远端 %s 个提交 —— 运行 tvop repo pull 获取最新知识库\n", behind)
	} else if ahead != "0" {
		fmt.Println("  ✓ 本地领先远端（有未推送的本地增量）")
	} else {
		fmt.Println("  ✓ 已与远端同步")
	}
	return exitOK
}

func repoPull(repo string) int {
	// 前置 1：工作区必须干净 —— 知识库是编译产物+手写概念混合体，
	// 带脏改动 pull 会把本地蒸馏成果卷进合并，属于不可自动裁决的冲突源。
	dirty, err := gitRun(repo, 5*time.Second, "status", "--porcelain")
	if err != nil {
		fmt.Fprintln(os.Stderr, "[invalid] repo.pull: 读取工作区状态失败：", err)
		return exitInvalid
	}
	if dirty != "" {
		fmt.Fprintln(os.Stderr, "[invalid] repo.pull: 工作区有未提交改动，拒绝自动拉取：")
		fmt.Fprintln(os.Stderr, dirty)
		fmt.Fprintln(os.Stderr, "  先提交或暂存（git stash）后再跑 tvop repo pull")
		return exitInvalid
	}
	branch, _ := gitRun(repo, 5*time.Second, "rev-parse", "--abbrev-ref", "HEAD")

	// 前置 2：只允许 fast-forward —— 不产生合并提交，不覆盖任何一方历史。
	if out, ferr := gitRun(repo, 60*time.Second, "pull", "--ff-only", "origin", branch); ferr != nil {
		fmt.Fprintln(os.Stderr, "[paid_failure] repo.pull: 拉取失败（网络或历史分叉）：")
		fmt.Fprintln(os.Stderr, out)
		fmt.Fprintln(os.Stderr, "  历史分叉时人工处置：git pull --rebase 或联系发送方")
		return exitInvalid
	}

	after, _ := gitRun(repo, 5*time.Second, "log", "-1", "--format=%h %ad %s", "--date=short")
	fmt.Println("tvop repo pull ✓")
	fmt.Println("  当前   :", after)
	fmt.Println("  收尾   : 知识库可能已更新，依次执行")
	fmt.Println("           tvop okf reindex && tvop okf validate && tvop govern recompute")
	return exitOK
}
