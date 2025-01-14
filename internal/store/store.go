package store

import (
	"context"
	"encoding/json"

	"github.com/pgvector/pgvector-go"

	"github.com/starbx/brew-api/pkg/ai"
	"github.com/starbx/brew-api/pkg/sqlstore"
	"github.com/starbx/brew-api/pkg/types"
)

// KnowledgeStoreInterface 定义 KnowledgeStore 的方法集合
type KnowledgeStore interface {
	sqlstore.SqlCommons
	// Create 创建新的知识记录
	Create(ctx context.Context, data types.Knowledge) error
	// GetKnowledge 根据ID获取知识记录
	GetKnowledge(ctx context.Context, spaceID, id string) (*types.Knowledge, error)
	// Update 更新知识记录
	Update(ctx context.Context, spaceID, id string, data types.UpdateKnowledgeArgs) error
	// Delete 删除知识记录
	Delete(ctx context.Context, spaceID, id string) error
	DeleteAll(ctx context.Context, spaceID string) error
	// ListKnowledges 分页获取知识记录列表
	ListKnowledges(ctx context.Context, opts types.GetKnowledgeOptions, page, pageSize uint64) ([]*types.Knowledge, error)
	Total(ctx context.Context, opts types.GetKnowledgeOptions) (uint64, error)
	ListLiteKnowledges(ctx context.Context, opts types.GetKnowledgeOptions, page, pageSize uint64) ([]*types.KnowledgeLite, error)
	FinishedStageSummarize(ctx context.Context, spaceID, id string, summary ai.ChunkResult) error
	FinishedStageEmbedding(ctx context.Context, spaceID, id string) error
	SetRetryTimes(ctx context.Context, spaceID, id string, retryTimes int) error
	ListProcessingKnowledges(ctx context.Context, retryTimes int, page, pageSize uint64) ([]types.Knowledge, error)
	ListFailedKnowledges(ctx context.Context, stage types.KnowledgeStage, retryTimes int, page, pageSize uint64) ([]types.Knowledge, error)
}

// KnowledgeChunkStore 定义 KnowledgeChunkStore 的接口
type KnowledgeChunkStore interface {
	sqlstore.SqlCommons
	Create(ctx context.Context, data types.KnowledgeChunk) error
	BatchCreate(ctx context.Context, data []types.KnowledgeChunk) error
	Get(ctx context.Context, spaceID, knowledgeID, id string) (*types.KnowledgeChunk, error)
	Update(ctx context.Context, spaceID, knowledgeID, id, chunk string) error
	Delete(ctx context.Context, spaceID, knowledgeID, id string) error
	BatchDelete(ctx context.Context, spaceID, knowledgeID string) error
	List(ctx context.Context, spaceID, knowledgeID string) ([]types.KnowledgeChunk, error)
}

// TODO support other vector db
// current only pg
// next qdrant
type VectorStore interface {
	sqlstore.SqlCommons
	Create(ctx context.Context, data types.Vector) error
	BatchCreate(ctx context.Context, datas []types.Vector) error
	GetVector(ctx context.Context, spaceID, knowledgeID, id string) (*types.Vector, error)
	Update(ctx context.Context, spaceID, knowledgeID, id string, vector pgvector.Vector) error
	Delete(ctx context.Context, spaceID, knowledgeID, id string) error
	BatchDelete(ctx context.Context, spaceID, knowledgeID string) error
	DeleteAll(ctx context.Context, spaceID string) error
	ListVectors(ctx context.Context, opts types.GetVectorsOptions, page, pageSize uint64) ([]types.Vector, error)
	Query(ctx context.Context, opts types.GetVectorsOptions, vectors pgvector.Vector, limit uint64) ([]types.QueryResult, error)
}

type AccessTokenStore interface {
	sqlstore.SqlCommons
	Create(ctx context.Context, data types.AccessToken) error
	GetAccessToken(ctx context.Context, appid, token string) (*types.AccessToken, error)
	Delete(ctx context.Context, appid, token string) error
	ListAccessTokens(ctx context.Context, appid, userID string, page, pageSize uint64) ([]types.AccessToken, error)
	ClearUserTokens(ctx context.Context, appid, userID string) error
}

type UserSpaceStore interface {
	sqlstore.SqlCommons
	Create(ctx context.Context, data types.UserSpace) error
	GetUserSpaceRole(ctx context.Context, userID, spaceID string) (*types.UserSpace, error)
	Update(ctx context.Context, userID, spaceID, role string) error
	List(ctx context.Context, opts types.ListUserSpaceOptions, page, pageSize uint64) ([]types.UserSpace, error)
	Total(ctx context.Context, opts types.ListUserSpaceOptions) (int64, error)
	Delete(ctx context.Context, userID, spaceID string) error
	DeleteAll(ctx context.Context, spaceID string) error
	ListSpaceUsers(ctx context.Context, spaceID string) ([]string, error)
}

type SpaceStore interface {
	sqlstore.SqlCommons
	Create(ctx context.Context, data types.Space) error
	GetSpace(ctx context.Context, spaceID string) (*types.Space, error)
	Update(ctx context.Context, spaceID, title, desc string) error
	Delete(ctx context.Context, spaceID string) error
	List(ctx context.Context, spaceIDs []string, page, pageSize uint64) ([]types.Space, error)
}

type ResourceStore interface {
	sqlstore.SqlCommons // 继承通用SQL操作
	Create(ctx context.Context, data types.Resource) error
	GetResource(ctx context.Context, spaceID, id string) (*types.Resource, error)
	Update(ctx context.Context, spaceID, id, title, desc, prompt string, cycle int) error
	Delete(ctx context.Context, spaceID, id string) error
	ListResources(ctx context.Context, spaceID string, page, pageSize uint64) ([]types.Resource, error)
}

type UserStore interface {
	sqlstore.SqlCommons // 继承通用SQL操作
	Create(ctx context.Context, data types.User) error
	GetUser(ctx context.Context, appid, id string) (*types.User, error)
	GetByEmail(ctx context.Context, appid, email string) (*types.User, error)
	UpdateUserProfile(ctx context.Context, appid, id, userName, email string) error
	Delete(ctx context.Context, appid, id string) error
	ListUsers(ctx context.Context, opts types.ListUserOptions, page, pageSize uint64) ([]types.User, error)
	Total(ctx context.Context, opts types.ListUserOptions) (int64, error)
}

type ChatSessionStore interface {
	sqlstore.SqlCommons // 继承通用SQL操作
	Create(ctx context.Context, data types.ChatSession) error
	UpdateSessionStatus(ctx context.Context, sessionID string, status types.ChatSessionStatus) error
	UpdateSessionTitle(ctx context.Context, sessionID string, title string) error
	GetByUserID(ctx context.Context, userID string) ([]*types.ChatSession, error)
	GetChatSession(ctx context.Context, spaceID, sessionID string) (*types.ChatSession, error)
	Delete(ctx context.Context, spaceID, sessionID string) error
	DeleteAll(ctx context.Context, spaceID string) error
	List(ctx context.Context, spaceID, userID string, page, pageSize uint64) ([]types.ChatSession, error)
	Total(ctx context.Context, spaceID, userID string) (int64, error)
}

type ChatMessageStore interface {
	sqlstore.SqlCommons // 继承通用SQL操作
	Create(ctx context.Context, data *types.ChatMessage) error
	GetOne(ctx context.Context, id string) (*types.ChatMessage, error)
	RewriteMessage(ctx context.Context, spaceID, sessionID, id string, message json.RawMessage, complete int32) error
	AppendMessage(ctx context.Context, spaceID, sessionID, id string, message json.RawMessage, complete int32) error
	UpdateMessageCompleteStatus(ctx context.Context, sessionID, id string, complete int32) error
	DeleteMessage(ctx context.Context, id string) error
	DeleteAll(ctx context.Context, spaceID string) error
	ListSessionMessageUpToGivenID(ctx context.Context, spaceID, sessionID, msgID string, page, pageSize uint64) ([]*types.ChatMessage, error)
	ListSessionMessage(ctx context.Context, spaceID, sessionID, afterMsgID string, page, pageSize uint64) ([]*types.ChatMessage, error)
	TotalSessionMessage(ctx context.Context, spaceID, sessionID, afterMsgID string) (int64, error)
	Exist(ctx context.Context, spaceID, sessionID, msgID string) (bool, error)
	GetMessagesByIDs(ctx context.Context, msgIDs []string) ([]*types.ChatMessage, error)
	GetSessionLatestMessage(ctx context.Context, spaceID, sessionID string) (*types.ChatMessage, error)
	GetSessionLatestUserMessage(ctx context.Context, spaceID, sessionID string) (*types.ChatMessage, error)
	GetSessionLatestUserMsgIDBeforeGivenID(ctx context.Context, spaceID, sessionID, msgID string) (string, error)
}

type ChatSummaryStore interface {
	sqlstore.SqlCommons
	Create(ctx context.Context, data types.ChatSummary) error
	GetChatSessionLatestSummary(ctx context.Context, sessionID string) (*types.ChatSummary, error)
}

type ChatMessageExtStore interface {
	sqlstore.SqlCommons // 假设你有通用的 SQL 操作接口
	Create(ctx context.Context, data types.ChatMessageExt) error
	GetChatMessageExt(ctx context.Context, spaceID, sessionID, messageID string) (*types.ChatMessageExt, error)
	ListChatMessageExts(ctx context.Context, messageIDs []string) ([]types.ChatMessageExt, error)
	Update(ctx context.Context, id string, data types.ChatMessageExt) error
	Delete(ctx context.Context, id string) error
	DeleteAll(ctx context.Context, spaceID string) error
}
