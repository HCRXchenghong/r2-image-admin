# 首次下载/使用项目时执行：自动安装 libvips + 安装前端依赖 + 准备数据目录
setup: install-vips
	cd frontend && npm install
	mkdir -p backend/data
	@echo "项目初始化完成。libvips 已就绪，前端依赖已安装。"

# 安装或校验项目强制依赖 libvips
install-vips:
	./scripts/install-libvips.sh

# 本地开发：后端（自动确保 libvips 已安装）
dev-api: install-vips
	cd backend && CGO_ENABLED=1 go run -tags=vips ./cmd/server

# 本地开发：前端（http://localhost:5173，自动代理到 8080）
dev-web:
	cd frontend && npm run dev

# 构建前端到 backend/web/dist
build-web:
	cd frontend && npm run build

# 编译后端（自动确保 libvips 已安装，并强制启用图片处理）
build: install-vips
	cd backend && CGO_ENABLED=1 go build -tags=vips -o bin/r2-image-admin ./cmd/server

test:
	cd backend && go vet ./... && go test ./...

docker:
	docker compose --env-file .env up --build -d
