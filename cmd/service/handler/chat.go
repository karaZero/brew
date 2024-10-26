package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	v1 "github.com/starbx/brew-api/internal/logic/v1"
	"github.com/starbx/brew-api/internal/response"
	"github.com/starbx/brew-api/pkg/errors"
	"github.com/starbx/brew-api/pkg/i18n"
	"github.com/starbx/brew-api/pkg/types"
	"github.com/starbx/brew-api/pkg/utils"
)

func (s *HttpSrv) GenMessageID(c *gin.Context) {
	response.APISuccess(c, s.Core.Srv().SeqSrv().GenMessageID())
}

type ListChatSessionRequest struct {
	Page     uint64 `json:"page" form:"page" binding:"required"`
	PageSize uint64 `json:"pagesize" form:"pagesize" binding:"required"`
}

type ListChatSessionResponse struct {
	List  []types.ChatSession `json:"list"`
	Total int64               `json:"total"`
}

func (s *HttpSrv) ListChatSession(c *gin.Context) {
	var (
		err error
		req ListChatSessionRequest
	)

	if err = utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	logic := v1.NewChatSessionLogic(c, s.Core)
	list, total, err := logic.ListUserChatSessions(req.Page, req.PageSize)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, ListChatSessionResponse{
		List:  list,
		Total: total,
	})
}

type RenameChatSessionRequest struct {
	FirstMessage string `json:"first_message" form:"first_message" binding:"required"`
}

func (s *HttpSrv) RenameChatSession(c *gin.Context) {
	sessionID, exist := c.Params.Get("session")
	if !exist || sessionID == "" {
		response.APIError(c, errors.New("api.DeleteChatSession", i18n.ERROR_INVALIDARGUMENT, nil).Code(http.StatusBadRequest))
		return
	}

	var (
		err error
		req RenameChatSessionRequest
	)
	if err = utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	logic := v1.NewChatSessionLogic(c, s.Core)

	space, _ := v1.InjectSpaceID(c)
	if _, err := logic.CheckUserChatSession(space, sessionID); err != nil {
		response.APIError(c, err)
		return
	}

	result, err := logic.NamedSession(sessionID, req.FirstMessage)
	if err != nil {
		response.APIError(c, err)
		return
	}
	response.APISuccess(c, result)
}

type CreateChatSessionResponse struct {
	SessionID string `json:"session_id"`
}

func (s *HttpSrv) CreateChatSession(c *gin.Context) {
	logic := v1.NewChatSessionLogic(c, s.Core)

	space, _ := v1.InjectSpaceID(c)
	sessionID, err := logic.CreateChatSession(space)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, CreateChatSessionResponse{
		SessionID: sessionID,
	})
}

func (s *HttpSrv) DeleteChatSession(c *gin.Context) {
	sessionID, exist := c.Params.Get("session")
	if !exist || sessionID == "" {
		response.APIError(c, errors.New("api.DeleteChatSession", i18n.ERROR_INVALIDARGUMENT, nil).Code(http.StatusBadRequest))
		return
	}
	logic := v1.NewChatSessionLogic(c, s.Core)

	space, _ := v1.InjectSpaceID(c)
	if _, err := logic.CheckUserChatSession(space, sessionID); err != nil {
		response.APIError(c, err)
		return
	}

	if err := logic.DeleteChatSession(space, sessionID); err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, nil)
}

type GetChatSessionHistoryRequest struct {
	Page           uint64 `json:"page" form:"page" binding:"required"`
	PageSize       uint64 `json:"pagesize" form:"pagesize" binding:"required"`
	AfterMessageID string `json:"after_message_id" form:"after_message_id"`
}

type GetChatSessionHistoryResponse struct {
	List  []*v1.MessageDetail `json:"list"`
	Total int64               `json:"total"`
}

func (s *HttpSrv) GetChatSessionHistory(c *gin.Context) {
	var (
		err error
		req GetChatSessionHistoryRequest
	)
	if err = utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	sessionID, exist := c.Params.Get("session")
	if !exist || sessionID == "" {
		response.APIError(c, errors.New("api.DeleteChatSession", i18n.ERROR_INVALIDARGUMENT, nil).Code(http.StatusBadRequest))
		return
	}

	sessionLogic := v1.NewChatSessionLogic(c, s.Core)

	space, _ := v1.InjectSpaceID(c)
	if _, err := sessionLogic.CheckUserChatSession(space, sessionID); err != nil {
		response.APIError(c, err)
		return
	}

	historyLogic := v1.NewHistoryLogic(c, s.Core)
	list, total, err := historyLogic.GetHistoryMessage(sessionID, req.AfterMessageID, req.Page, req.PageSize)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, GetChatSessionHistoryResponse{
		List:  lo.Reverse(list),
		Total: total,
	})
}

type CreateChatMessageRequest struct {
	MessageID string               `json:"message_id" binding:"required"`
	Message   string               `json:"message" binding:"required"`
	Resource  *types.ResourceQuery `json:"resource"`
}

type CreateChatMessageResponse struct {
	Sequence int64 `json:"sequence"`
}

func (s *HttpSrv) CreateChatMessage(c *gin.Context) {
	var (
		err error
		req CreateChatMessageRequest
	)

	if err = utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	sessionID, exist := c.Params.Get("session")
	if !exist || sessionID == "" {
		response.APIError(c, errors.New("api.DeleteChatSession", i18n.ERROR_INVALIDARGUMENT, nil).Code(http.StatusBadRequest))
		return
	}

	sessionLogic := v1.NewChatSessionLogic(c, s.Core)
	space, _ := v1.InjectSpaceID(c)
	session, err := sessionLogic.CheckUserChatSession(space, sessionID)
	if err != nil {
		response.APIError(c, err)
		return
	}

	chatLogic := v1.NewChatLogic(c, s.Core)
	msgSequence, err := chatLogic.NewUserMessage(session, types.CreateChatMessageArgs{
		ID:       req.MessageID,
		Message:  req.Message,
		MsgType:  types.MESSAGE_TYPE_TEXT,
		SendTime: time.Now().Unix(),
	}, req.Resource)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, CreateChatMessageResponse{
		Sequence: msgSequence,
	})
}

func (s *HttpSrv) GetChatMessageExt(c *gin.Context) {
	sessionID, _ := c.Params.Get("session")
	messageID, _ := c.Params.Get("messageid")

	sessionLogic := v1.NewChatSessionLogic(c, s.Core)
	space, _ := v1.InjectSpaceID(c)
	_, err := sessionLogic.CheckUserChatSession(space, sessionID)
	if err != nil {
		response.APIError(c, err)
		return
	}

	ext, err := v1.NewHistoryLogic(c, s.Core).GetMessageExt(sessionID, messageID)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, ext)
}