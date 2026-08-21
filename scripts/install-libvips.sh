#!/usr/bin/env bash
set -euo pipefail

# 项目强制依赖 libvips：本地运行智能处理、生成 WebP/AVIF 变体都需要它。
# 该脚本会按操作系统自动安装 libvips，已安装时直接跳过。

if pkg-config --exists vips 2>/dev/null; then
  echo "[install-libvips] libvips 已安装，跳过。"
  exit 0
fi

OS="$(uname -s)"
echo "[install-libvips] 检测到系统：$OS"

case "$OS" in
  Darwin)
    if ! command -v brew >/dev/null 2>&1; then
      echo "[install-libvips] 错误：请先安装 Homebrew（https://brew.sh）。" >&2
      exit 1
    fi
    echo "[install-libvips] 正在通过 Homebrew 安装 libvips..."
    if ! HOMEBREW_NO_AUTO_UPDATE=1 brew install vips; then
      echo "[install-libvips] brew install 返回非零，继续检查 libvips 是否实际可用。" >&2
    fi
    ;;

  Linux)
    if command -v apt-get >/dev/null 2>&1; then
      echo "[install-libvips] 正在通过 apt-get 安装 libvips-dev..."
      if [ "$(id -u)" = "0" ]; then
        apt-get update && apt-get install -y libvips-dev pkg-config
      else
        sudo apt-get update && sudo apt-get install -y libvips-dev pkg-config
      fi
    elif command -v apk >/dev/null 2>&1; then
      echo "[install-libvips] 正在通过 apk 安装 vips-dev..."
      if [ "$(id -u)" = "0" ]; then
        apk add --no-cache vips-dev pkgconfig
      else
        sudo apk add --no-cache vips-dev pkgconfig
      fi
    elif command -v dnf >/dev/null 2>&1; then
      echo "[install-libvips] 正在通过 dnf 安装 vips-devel..."
      if [ "$(id -u)" = "0" ]; then
        dnf install -y vips-devel pkg-config
      else
        sudo dnf install -y vips-devel pkg-config
      fi
    elif command -v yum >/dev/null 2>&1; then
      echo "[install-libvips] 正在通过 yum 安装 vips-devel..."
      if [ "$(id -u)" = "0" ]; then
        yum install -y vips-devel pkg-config
      else
        sudo yum install -y vips-devel pkg-config
      fi
    else
      echo "[install-libvips] 错误：未识别 Linux 包管理器，请手动安装 libvips-dev / vips-dev。" >&2
      exit 1
    fi
    ;;

  *)
    echo "[install-libvips] 错误：暂不支持 $OS，请手动安装 libvips。" >&2
    exit 1
    ;;
esac

if ! pkg-config --exists vips 2>/dev/null; then
  echo "[install-libvips] 安装后仍检测不到 libvips，请检查 pkg-config 配置。" >&2
  exit 1
fi

echo "[install-libvips] libvips 安装完成。"
