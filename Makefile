REGISTRY ?= ghcr.io/hwameistor
GO_VERSION = $(shell go version)
BUILD_TIME = ${shell date +%Y-%m-%dT%H:%M:%SZ}
BUILD_VERSION = ${shell git rev-parse --short "HEAD^{commit}" 2>/dev/null}
BUILD_ENVS = CGO_ENABLED=0 GOOS=linux
BUILD_FLAGS = -X 'main.BUILDVERSION=${BUILD_VERSION}' -X 'main.BUILDTIME=${BUILD_TIME}' -X 'main.GOVERSION=${GO_VERSION}'
BUILD_OPTIONS = -a -mod vendor -installsuffix cgo -ldflags "${BUILD_FLAGS}"
PROJECT_SOURCE_CODE_DIR=$(CURDIR)
BINS_DIR = ${PROJECT_SOURCE_CODE_DIR}/_build
CMDS_DIR = ${PROJECT_SOURCE_CODE_DIR}/cmd
IMAGES_DIR = ${PROJECT_SOURCE_CODE_DIR}/images
BUILD_CMD = go build
BUILDER_NAME = ${REGISTRY}/drbd-installer-builder
BUILDER_IMAGE_TAG = latest
BUILDER_MOUNT_SRC_DIR = ${PROJECT_SOURCE_CODE_DIR}/../
BUILDER_MOUNT_DST_DIR = /go/src/github.com/hwameistor
BUILDER_WORKDIR = /go/src/github.com/hwameistor/drbd-installer
DOCKER_SOCK_PATH=/var/run/docker.sock
DOCKER_MAKE_CMD = docker run --rm -v ${BUILDER_MOUNT_SRC_DIR}:${BUILDER_MOUNT_DST_DIR} -v ${DOCKER_SOCK_PATH}:${DOCKER_SOCK_PATH} -w ${BUILDER_WORKDIR} -i ${BUILDER_NAME}:${BUILDER_IMAGE_TAG}
DOCKER_BUILDX_CMD_AMD64 = DOCKER_CLI_EXPERIMENTAL=enabled docker buildx build --platform=linux/amd64 -o type=docker
DOCKER_BUILDX_CMD_ARM64 = DOCKER_CLI_EXPERIMENTAL=enabled docker buildx build --platform=linux/arm64 -o type=docker
MUILT_ARCH_PUSH_CMD = ${PROJECT_SOURCE_CODE_DIR}/docker-push-with-multi-arch.sh
IMAGE_TAG ?= v99.9.9
DRBD_INSTALLER_NAME = drbd-installer
DRBD_INSTALLER_IMAGE_DIR = ${PROJECT_SOURCE_CODE_DIR}/build
DRBD_INSTALLER_BUILD_BIN = ${BINS_DIR}/${DRBD_INSTALLER_NAME}
DRBD_INSTALLER_BUILD_MAIN = ${CMDS_DIR}/main.go
DRBD_INSTALLER_IMAGE_NAME = ${REGISTRY}/${DRBD_INSTALLER_NAME}

.PHONY: builder
builder:
	docker build -t ${BUILDER_NAME}:${BUILDER_IMAGE_TAG} -f build/builder/Dockerfile .
	docker push ${BUILDER_NAME}:${BUILDER_IMAGE_TAG}

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor
	
.PHONY: drbd_installer
drbd_installer:
	GOARCH=amd64 ${BUILD_ENVS} ${BUILD_CMD} ${BUILD_OPTIONS} -o ${DRBD_INSTALLER_BUILD_BIN} ${DRBD_INSTALLER_BUILD_MAIN}

.PHONY: drbd_installer_arm64
drbd_installer_arm64:
	GOARCH=arm64 ${BUILD_ENVS} ${BUILD_CMD} ${BUILD_OPTIONS} -o ${DRBD_INSTALLER_BUILD_BIN} ${DRBD_INSTALLER_BUILD_MAIN}

.PHONY: drbd_installer_image
drbd_installer_image:
	${DOCKER_MAKE_CMD} make drbd_installer
	docker build -t ${DRBD_INSTALLER_IMAGE_NAME}:${IMAGE_TAG} -f ${DRBD_INSTALLER_IMAGE_DIR}/Dockerfile ${PROJECT_SOURCE_CODE_DIR}

.PHONY: release
release:
	# build for amd64 version
	${DOCKER_MAKE_CMD} make drbd_installer
	${DOCKER_BUILDX_CMD_AMD64} -t ${DRBD_INSTALLER_IMAGE_NAME}:${IMAGE_TAG}-amd64 -f ${DRBD_INSTALLER_IMAGE_DIR}/Dockerfile ${PROJECT_SOURCE_CODE_DIR}
	# build for arm64 version
	${DOCKER_MAKE_CMD} make drbd_installer_arm64
	${DOCKER_BUILDX_CMD_ARM64} -t ${DRBD_INSTALLER_IMAGE_NAME}:${IMAGE_TAG}-arm64 -f ${DRBD_INSTALLER_IMAGE_DIR}/Dockerfile ${PROJECT_SOURCE_CODE_DIR}
	# push to a public registry
	${MUILT_ARCH_PUSH_CMD} ${DRBD_INSTALLER_IMAGE_NAME}:${IMAGE_TAG}
