#!/bin/bash
# 图片生成功能诊断脚本

echo "=========================================="
echo "图片生成功能诊断"
echo "=========================================="
echo ""

echo "1. 检查容器状态"
echo "----------------------------------------"
docker ps | grep sub2api
echo ""

echo "2. 检查 Python 脚本是否存在"
echo "----------------------------------------"
docker exec sub2api-plus-sub2api-1 ls -lh /app/chatgpt_image_proxy.py 2>&1
echo ""

echo "3. 检查 Python 环境"
echo "----------------------------------------"
docker exec sub2api-plus-sub2api-1 python3 --version 2>&1
docker exec sub2api-plus-sub2api-1 python3 -c "import curl_cffi; print('curl_cffi 版本:', curl_cffi.__version__)" 2>&1
echo ""

echo "4. 查看最近的图片生成日志（最近 50 行）"
echo "----------------------------------------"
docker logs sub2api-plus-sub2api-1 2>&1 | grep -i "chatgpt_image\|image.*generate" | tail -50
echo ""

echo "5. 查看最近的错误日志（最近 20 行）"
echo "----------------------------------------"
docker logs sub2api-plus-sub2api-1 2>&1 | grep -i "error\|failed\|panic" | tail -20
echo ""

echo "=========================================="
echo "诊断完成"
echo "=========================================="
echo ""
echo "请将以上输出发送给开发者进行分析"
