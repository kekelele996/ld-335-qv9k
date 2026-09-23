package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/blueship581/gbinsureapi/internal/constants"
	"github.com/blueship581/gbinsureapi/internal/model"
	"github.com/blueship581/gbinsureapi/internal/repository"
	"github.com/blueship581/gbinsureapi/internal/util"
)

// ApiClientService 调用方服务：注册、权限、API Key 校验与限流配置。
type ApiClientService struct {
	repo         *repository.ApiClientRepository
	apiKeySecret string
	jwtSecret    string
	jwtExpire    int
	log          *slog.Logger
}

// NewApiClientService 构造调用方服务。
func NewApiClientService(repo *repository.ApiClientRepository, apiKeySecret, jwtSecret string, jwtExpire int, log *slog.Logger) *ApiClientService {
	return &ApiClientService{repo: repo, apiKeySecret: apiKeySecret, jwtSecret: jwtSecret, jwtExpire: jwtExpire, log: log}
}

// CreateClientInput 创建调用方入参。
type CreateClientInput struct {
	Name         string `json:"name" binding:"required,max=100"`
	ClientType   string `json:"client_type" binding:"required,oneof=HIS THIRD_PARTY"`
	Role         string `json:"role" binding:"required"`
	RateLimitQPS int    `json:"rate_limit_qps" binding:"gte=1,lte=1000"`
}

// CreateClient 创建调用方，返回明文 API Key（仅此一次）。
func (s *ApiClientService) CreateClient(ctx context.Context, input CreateClientInput) (*model.ApiClient, string, error) {
	apiKey, err := util.GenerateAPIKey()
	if err != nil {
		return nil, "", util.LogError(s.log, constants.LOG_CLIENT_CREATED, fmt.Errorf("generate api key: %w", err))
	}
	qps := input.RateLimitQPS
	if qps <= 0 {
		qps = 10
	}
	client := &model.ApiClient{
		Name: input.Name, ClientType: input.ClientType,
		APIKeyHash: util.HashAPIKey(apiKey, s.apiKeySecret),
		Role:       input.Role, Status: constants.ClientActive, RateLimitQPS: qps,
	}
	if err := s.repo.Create(client); err != nil {
		return nil, "", util.LogError(s.log, constants.LOG_CLIENT_CREATED, fmt.Errorf("create client: %w", err))
	}
	s.log.InfoContext(ctx, constants.LOG_CLIENT_CREATED, "client_id", client.ID, "name", client.Name)
	return client, apiKey, nil
}

// VerifyAPIKey 校验 API Key 并返回调用方。
func (s *ApiClientService) VerifyAPIKey(ctx context.Context, apiKey string) (*model.ApiClient, error) {
	clients, total, err := s.repo.List(1, 1000)
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, util.UnauthorizedError(constants.MsgApiKeyInvalid, util.ErrApiKeyInvalid)
	}
	for _, c := range clients {
		if util.VerifyAPIKey(apiKey, c.APIKeyHash, s.apiKeySecret) {
			if c.Status != constants.ClientActive {
				return nil, util.ForbiddenError(constants.MsgClientDisabled, util.ErrClientDisabled)
			}
			return &c, nil
		}
	}
	return nil, util.UnauthorizedError(constants.MsgApiKeyInvalid, util.ErrApiKeyInvalid)
}

// IssueServiceToken 为调用方签发服务 JWT（API Key 已校验后）。
func (s *ApiClientService) IssueServiceToken(ctx context.Context, client *model.ApiClient) (string, error) {
	token, err := util.GenerateToken(s.jwtSecret, s.jwtExpire, client.ID, client.Name, client.Role, "service")
	if err != nil {
		return "", util.LogError(s.log, constants.LOG_CLIENT_TOKEN_FAILED, fmt.Errorf("issue token: %w", err))
	}
	s.log.InfoContext(ctx, constants.LOG_CLIENT_TOKEN_ISSUED, "client_id", client.ID)
	return token, nil
}

// List 分页查询调用方。
func (s *ApiClientService) List(ctx context.Context, page, pageSize int) ([]model.ApiClient, int64, error) {
	return s.repo.List(page, pageSize)
}

// UpdateStatus 启用/停用调用方。
func (s *ApiClientService) UpdateStatus(ctx context.Context, id uint, status string) error {
	client, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			return util.NotFoundError("调用方（ApiClient）不存在", err)
		}
		return util.LogError(s.log, constants.LOG_CLIENT_STATUS_CHANGED, fmt.Errorf("find client: %w", err))
	}
	if err := s.repo.UpdateStatus(client.ID, status); err != nil {
		return util.LogError(s.log, constants.LOG_CLIENT_STATUS_CHANGED, fmt.Errorf("update status: %w", err))
	}
	s.log.InfoContext(ctx, constants.LOG_CLIENT_STATUS_CHANGED, "client_id", id, "status", status)
	return nil
}
