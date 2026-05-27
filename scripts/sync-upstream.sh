#!/bin/bash
# Sub2API Plus 上游同步检查脚本
# 用法：bash scripts/sync-upstream.sh

set -e

REPO_URL="https://github.com/Wei-Shaw/sub2api.git"
SYNC_FILE=".upstream-sync"

# 读取当前同步的 commit hash
LAST_HASH=$(grep "^COMMIT_HASH=" "$SYNC_FILE" | cut -d= -f2)

echo "========================================="
echo "Sub2API Plus 上游同步检查"
echo "========================================="
echo "上次同步 commit: $LAST_HASH"
echo "上游仓库: $REPO_URL"
echo ""

# 拉取上游最新
echo "拉取上游最新 commits..."
HTTPS_PROXY="${HTTPS_PROXY:-}" git fetch upstream --quiet 2>/dev/null || true
git fetch upstream --quiet

# 获取当前上游 HEAD
UPSTREAM_HEAD=$(git rev-parse upstream/main)
UPSTREAM_HEAD_MSG=$(git log upstream/main -1 --oneline)
echo "上游最新: $UPSTREAM_HEAD_MSG"
echo ""

# 计算新 commits 数量
NEW_COMMITS=$(git rev-list "$LAST_HASH"..upstream/main --count 2>/dev/null || echo "0")

if [ "$NEW_COMMITS" = "0" ]; then
    echo "✅ 已是最新，无新变更"
else
    echo "📢 有 $NEW_COMMITS 个新 commit："
    echo ""
    git log upstream/main --oneline "$LAST_HASH"..upstream/main | head -20
    if [ "$NEW_COMMITS" -gt 20 ]; then
        echo "... 还有 $((NEW_COMMITS - 20)) 个"
    fi
    echo ""
    echo "查看完整变更：git log upstream/main --not main"
    echo "查看文件差异：git diff main upstream/main --stat"
fi

echo ""
echo "更新同步记录："
echo "  修改 .upstream-sync 中的 COMMIT_HASH=$UPSTREAM_HEAD"
echo "========================================="
