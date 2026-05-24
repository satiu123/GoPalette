GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)

ifeq ($(GOHOSTOS), windows)
	#the `find.exe` is different from `find` in bash/shell.
	#to see https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/find.
	#changed to use git-bash.exe to run find cli or other cli friendly, caused of every developer has a Git.
	#Git_Bash= $(subst cmd\,bin\bash.exe,$(dir $(shell where git)))
	Git_Bash=$(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
	SERVICE_PROTO_FILES=$(shell $(Git_Bash) -c "find app -name '*.proto'")
	API_PROTO_FILES=$(shell $(Git_Bash) -c "find api -name '*.proto'")
else
	SERVICE_PROTO_FILES=$(shell find app -name '*.proto')
	API_PROTO_FILES=$(shell find api -name '*.proto')
endif

.PHONY: init
# init env
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/google/wire/cmd/wire@latest

.PHONY: config
# generate internal proto
config:
	protoc --proto_path=. \
	       --proto_path=./third_party \
	       --go_out=paths=source_relative:. \
	       $(SERVICE_PROTO_FILES)

.PHONY: api
# generate api proto
api:
	protoc --proto_path=./api \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:./api \
 	       --go-http_out=paths=source_relative:./api \
 	       --go-grpc_out=paths=source_relative:./api \
	       --openapi_out=fq_schema_naming=true,default_response=false:. \
	       $(API_PROTO_FILES)

.PHONY: errors
# generate errors code
errors:
	protoc --proto_path=. \
           --proto_path=./third_party \
           --go_out=paths=source_relative:. \
           --go-errors_out=paths=source_relative:. \
           $(API_PROTO_FILES)


.PHONY: build
# build
build:
	mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./...

.PHONY: test
# run workspace tests
test:
	go test work

.PHONY: generate
# generate
generate:
	go generate work
	go work sync

.PHONY: all
# generate all
all:
	$(MAKE) api
	$(MAKE) config
	$(MAKE) generate

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help


define HEALTH_CHECKER_GO
package health

import "context"

type Checker interface {
	Name() string
	Check(context.Context) error
}
endef
export HEALTH_CHECKER_GO

define HEALTH_HTTP_GO
package health

import (
	"context"
	"net/http"
	"time"
)

func RegisterHTTP(
	mux *http.ServeMux,
	h *Health,
) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()

		if err := h.Check(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}
endef
export HEALTH_HTTP_GO

define HEALTH_GRPC_GO
package health

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type grpcServer struct {
	grpc_health_v1.UnimplementedHealthServer
	h *Health
}

func RegisterGRPC(
	srv grpc.ServiceRegistrar,
	h *Health,
) {
	grpc_health_v1.RegisterHealthServer(srv, &grpcServer{h: h})
}

func (s *grpcServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	if err := s.h.Check(ctx); err != nil {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		}, nil
	}
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

func (s *grpcServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "unimplemented")
}
endef
export HEALTH_GRPC_GO

define HEALTH_MYSQL_GO
package health

import (
	"context"

	"gorm.io/gorm"
)

type MySQLChecker struct {
	db *gorm.DB
}

func NewMySQLChecker(db *gorm.DB) *MySQLChecker {
	return &MySQLChecker{
		db: db,
	}
}

func (c *MySQLChecker) Name() string {
	return "mysql"
}

func (c *MySQLChecker) Check(ctx context.Context) error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return err
	}

	return sqlDB.PingContext(ctx)
}
endef
export HEALTH_MYSQL_GO

define HEALTH_REDIS_GO
package health

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisChecker struct {
	rdb *redis.Client
}

func NewRedisChecker(rdb *redis.Client) *RedisChecker {
	return &RedisChecker{
		rdb: rdb,
	}
}

func (c *RedisChecker) Name() string {
	return "redis"
}

func (c *RedisChecker) Check(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}
endef
export HEALTH_REDIS_GO


comma := ,
empty :=
space := $(empty) $(empty)
CLEANED_CHECKERS = $(subst $(comma),$(space),$(CHECKERS))

# 检测具体启用了哪些 Checkers
MYSQL_ENABLED := $(filter mysql,$(CLEANED_CHECKERS))
REDIS_ENABLED := $(filter redis,$(CLEANED_CHECKERS))

# 1. 动态生成 Import 部分（无 Checker 时无需引入 internal/data）
define HEALTH_IMPORTS_WITH_DATA
import (
	"context"
	"fmt"

	"github.com/euskadi31/wire"
	"github.com/satiu123/GoPalette/app/$(SERVICE)/service/internal/data"
)
endef

define HEALTH_IMPORTS_NO_DATA
import (
	"context"
	"fmt"

	"github.com/euskadi31/wire"
)
endef

HEALTH_IMPORTS = $(if $(or $(MYSQL_ENABLED),$(REDIS_ENABLED)),$(HEALTH_IMPORTS_WITH_DATA),$(HEALTH_IMPORTS_NO_DATA))

# 2. 动态生成 ProviderSet 列表
HEALTH_PROVIDER_SET_STR = $(if $(and $(MYSQL_ENABLED),$(REDIS_ENABLED)),NewMySQLCheckerWrapper$(comma) NewRedisCheckerWrapper$(comma) NewHealthWrapper,$(if $(MYSQL_ENABLED),NewMySQLCheckerWrapper$(comma) NewHealthWrapper,$(if $(REDIS_ENABLED),NewRedisCheckerWrapper$(comma) NewHealthWrapper,NewHealthWrapper)))

# 3. 动态生成 MySQL/Redis 专属的 Wrapper 桥接函数
define MYSQL_WRAPPERS_TEMPLATE
// 桥接 Data 内部的 DB 实例
func NewMySQLCheckerWrapper(d *data.Data) *MySQLChecker {
	return NewMySQLChecker(d.DB())
}
endef

define REDIS_WRAPPERS_TEMPLATE
// 桥接 Data 内部的 Redis 实例
func NewRedisCheckerWrapper(d *data.Data) *RedisChecker {
	return NewRedisChecker(d.Redis())
}
endef

MYSQL_WRAPPERS = $(if $(MYSQL_ENABLED),$(MYSQL_WRAPPERS_TEMPLATE))
REDIS_WRAPPERS = $(if $(REDIS_ENABLED),$(REDIS_WRAPPERS_TEMPLATE))

# 4. 动态生成 NewHealthWrapper 构造函数签名
define NEW_HEALTH_WRAPPER_BOTH
// 组合并生成总的 Health 检查器
func NewHealthWrapper(mysql *MySQLChecker, redis *RedisChecker) *Health {
	return New(mysql, redis)
}
endef

define NEW_HEALTH_WRAPPER_MYSQL
// 组合并生成总的 Health 检查器
func NewHealthWrapper(mysql *MySQLChecker) *Health {
	return New(mysql)
}
endef

define NEW_HEALTH_WRAPPER_REDIS
// 组合并生成总的 Health 检查器
func NewHealthWrapper(redis *RedisChecker) *Health {
	return New(redis)
}
endef

define NEW_HEALTH_WRAPPER_NONE
// 组合并生成总的 Health 检查器
func NewHealthWrapper() *Health {
	return New()
}
endef

NEW_HEALTH_WRAPPER = $(if $(and $(MYSQL_ENABLED),$(REDIS_ENABLED)),$(NEW_HEALTH_WRAPPER_BOTH),$(if $(MYSQL_ENABLED),$(NEW_HEALTH_WRAPPER_MYSQL),$(if $(REDIS_ENABLED),$(NEW_HEALTH_WRAPPER_REDIS),$(NEW_HEALTH_WRAPPER_NONE))))

# 5. 组装最终的 HEALTH_GO 模板
define HEALTH_GO
package health

$(HEALTH_IMPORTS)

var ProviderSet = wire.NewSet($(HEALTH_PROVIDER_SET_STR))

$(MYSQL_WRAPPERS)

$(REDIS_WRAPPERS)

$(NEW_HEALTH_WRAPPER)

type Health struct {
	checkers []Checker
}

func New(cs ...Checker) *Health {
	return &Health{
		checkers: cs,
	}
}

func (h *Health) Check(ctx context.Context) error {
	var errs []error
	for _, c := range h.checkers {
		if err := c.Check(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", c.Name(), err))
		}
	}

	// errors.Join 如果传入空切片会安全返回 nil，否则合并所有非空错误
	return errors.Join(errs...) 
}
endef
export HEALTH_GO


.PHONY: health
# create health structure, usage: make health SERVICE=user CHECKERS="mysql redis"
health:
ifndef SERVICE
	$(error SERVICE is required. usage: make health SERVICE=user CHECKERS="mysql redis")
endif

	@echo "==> creating health structure for $(SERVICE)"

	@mkdir -p app/$(SERVICE)/service/internal/health

	@printf "%s\n" "$$HEALTH_CHECKER_GO" > app/$(SERVICE)/service/internal/health/checker.go
	@printf "%s\n" "$$HEALTH_GO" > app/$(SERVICE)/service/internal/health/health.go
	@printf "%s\n" "$$HEALTH_HTTP_GO" > app/$(SERVICE)/service/internal/health/http.go
	@printf "%s\n" "$$HEALTH_GRPC_GO" > app/$(SERVICE)/service/internal/health/grpc.go

ifneq ($(filter mysql,$(CLEANED_CHECKERS)),)
	@printf "%s\n" "$$HEALTH_MYSQL_GO" > app/$(SERVICE)/service/internal/health/mysql.go
	@echo "==> [mysql] checker added successfully"
endif

ifneq ($(filter redis,$(CLEANED_CHECKERS)),)
	@printf "%s\n" "$$HEALTH_REDIS_GO" > app/$(SERVICE)/service/internal/health/redis.go
	@echo "==> [redis] checker added successfully"
endif

	@echo "==> health structure created successfully"