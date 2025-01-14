package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/starbx/brew-api/internal/core"
	v1 "github.com/starbx/brew-api/internal/logic/v1"
	"github.com/starbx/brew-api/internal/response"
	"github.com/starbx/brew-api/pkg/types"
	"github.com/starbx/brew-api/pkg/utils"
)

type HttpSrv struct {
	Core   *core.Core
	Engine *gin.Engine
}

type UpdateKnowledgeRequest struct {
	ID       string              `json:"id" binding:"required"`
	Title    string              `json:"title"`
	Resource string              `json:"resource"`
	Content  string              `json:"content"`
	Tags     []string            `json:"tags"`
	Kind     types.KnowledgeKind `json:"kind"`
}

func (s *HttpSrv) UpdateKnowledge(c *gin.Context) {
	var (
		err error
		req UpdateKnowledgeRequest
	)
	if err = utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	spaceID, _ := v1.InjectSpaceID(c)
	err = v1.NewKnowledgeLogic(c, s.Core).Update(spaceID, req.ID, types.UpdateKnowledgeArgs{
		Title:    req.Title,
		Content:  req.Content,
		Resource: req.Resource,
		Tags:     req.Tags,
		Kind:     req.Kind,
	})
	if err != nil {
		response.APIError(c, err)
		return
	}
	response.APISuccess(c, nil)
}

type CreateKnowledgeRequest struct {
	Resource string `json:"resource"`
	Content  string `json:"content" binding:"required"`
	Kind     string `json:"kind"`
	Async    bool   `json:"async"`
}

type CreateKnowledgeResponse struct {
	ID string `json:"id"`
}

func (s *HttpSrv) CreateKnowledge(c *gin.Context) {
	var req CreateKnowledgeRequest

	if err := utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	spaceID, _ := v1.InjectSpaceID(c)
	var handler func(spaceID, resource string, kind types.KnowledgeKind, content string) (string, error)
	logic := v1.NewKnowledgeLogic(c, s.Core)
	if req.Async {
		handler = logic.InsertContentAsync
	} else {
		handler = logic.InsertContent
	}
	id, err := handler(spaceID, req.Resource, types.KindNewFromString(req.Kind), req.Content)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, CreateKnowledgeResponse{
		ID: id,
	})
}

type GetKnowledgeRequest struct {
	ID string `json:"id" form:"id" binding:"required"`
}

func (s *HttpSrv) GetKnowledge(c *gin.Context) {
	var (
		err error
		req GetKnowledgeRequest
	)
	if err = utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	spaceID, _ := v1.InjectSpaceID(c)
	knowledge, err := v1.NewKnowledgeLogic(c, s.Core).GetKnowledge(spaceID, req.ID)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, knowledge)
}

type ListKnowledgeRequest struct {
	Resource string `json:"resource" form:"resource"`
	Page     uint64 `json:"page" form:"page" binding:"required"`
	PageSize uint64 `json:"pagesize" form:"pagesize" binding:"required,lte=50"`
}

type ListKnowledgeResponse struct {
	List  []*types.Knowledge `json:"list"`
	Total uint64             `json:"total"`
}

func (s *HttpSrv) ListKnowledge(c *gin.Context) {
	var req ListKnowledgeRequest

	if err := utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	var resource *types.ResourceQuery
	if req.Resource != "" {
		resource = &types.ResourceQuery{
			Include: []string{req.Resource},
		}
	}

	spaceID, _ := v1.InjectSpaceID(c)
	list, total, err := v1.NewKnowledgeLogic(c, s.Core).ListKnowledges(spaceID, resource, req.Page, req.PageSize)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, ListKnowledgeResponse{
		List:  list,
		Total: total,
	})
}

type DeleteKnowledgeRequest struct {
	ID string `json:"id" binding:"required"`
}

func (s *HttpSrv) DeleteKnowledge(c *gin.Context) {
	var req DeleteKnowledgeRequest
	if err := utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	spaceID, _ := v1.InjectSpaceID(c)
	if err := v1.NewKnowledgeLogic(c, s.Core).Delete(spaceID, req.ID); err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, nil)
}

type QueryRequest struct {
	Query    string               `json:"query" binding:"required"`
	Resource *types.ResourceQuery `json:"resource"`
}

func (s *HttpSrv) Query(c *gin.Context) {
	var req QueryRequest

	if err := utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	spaceID, _ := v1.InjectSpaceID(c)
	// v1.KnowledgeQueryResult
	result, err := v1.NewKnowledgeLogic(c, s.Core).Query(spaceID, req.Resource, req.Query)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, result)
}
