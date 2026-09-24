#!/bin/bash
# 本地交叉编译打包脚本，与 .github/workflows/build-release.yml 保持一致。
# 产物命名：imapdownloader_<version>_<os>_<arch>.{tar.gz|zip}
# 版本号：tag 检出用 tag 名，否则 dev-<短SHA>（与 CI 一致）
set -eu

cd "$(dirname "$0")"
rm -rf dist stage
mkdir -p dist stage

# 版本号确定：优先 git 当前 tag，无 tag 用 dev-<短SHA>
if git describe --tags --exact-match >/dev/null 2>&1; then
  VERSION="$(git describe --tags --exact-match)"
else
  VERSION="dev-$(git rev-parse --short HEAD)"
fi
echo "==> version: ${VERSION}"

# 选择可用的 sha256 工具（Linux: sha256sum，macOS: shasum -a 256）
SHA_CMD="sha256sum"
command -v sha256sum >/dev/null 2>&1 || SHA_CMD="shasum -a 256"

# build <goos> <goarch> [goarm] <archive(tar.gz|zip)>
build() {
  local goos=$1 goarch=$2 goarm=${3:-} archive=$4
  export GOOS="$goos" GOARCH="$goarch" GOARM="$goarm" CGO_ENABLED=0
  local arch="$goarch"
  [[ -n "$goarm" ]] && arch="${goarch}v${goarm}"

  local stage_dir="stage/imapdownloader_${VERSION}_${goos}_${arch}"
  mkdir -p "$stage_dir/imapdownloader"
  local bin="imapdownloader"
  [[ "$goos" == "windows" ]] && bin="imapdownloader.exe"

  # -trimpath 去绝对路径；-s -w 去 symbol/table 减体积；-X 注入版本到 main.version
  go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
    -o "$stage_dir/imapdownloader/$bin" .
  cp LICENSE README.md config.yaml "$stage_dir/imapdownloader/"

  local out="dist/imapdownloader_${VERSION}_${goos}_${arch}"
  if [[ "$archive" == "zip" ]]; then
    (cd "$stage_dir" && zip -rq "../../${out}.zip" imapdownloader)
    echo "✓ ${out}.zip"
  else
    tar -C "$stage_dir" -czf "${out}.tar.gz" imapdownloader
    echo "✓ ${out}.tar.gz"
  fi
}

build linux   amd64   ""  tar.gz
build linux   arm64   ""  tar.gz
build linux   386     ""  tar.gz
build linux   arm     7   tar.gz
build linux   loong64 ""  tar.gz
build darwin  amd64   ""  tar.gz
build darwin  arm64   ""  tar.gz
build windows amd64   ""  zip
build windows arm64   ""  zip
build windows 386     ""  zip

# 生成 sha256 校验文件
( cd dist && $SHA_CMD *.tar.gz *.zip > checksums.txt )
echo "✓ dist/checksums.txt"
echo "==> done. artifacts:"
ls -la dist
