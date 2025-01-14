package handler

import (
	"github.com/gin-gonic/gin"

	v1 "github.com/starbx/brew-api/internal/logic/v1"
	"github.com/starbx/brew-api/internal/response"
	"github.com/starbx/brew-api/pkg/types"
	"github.com/starbx/brew-api/pkg/utils"
)

type ListUserSpacesResponse struct {
	List []types.UserSpaceDetail `json:"list"`
}

func (s *HttpSrv) ListUserSpaces(c *gin.Context) {
	list, err := v1.NewSpaceLogic(c, s.Core).ListUserSpace()
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, ListUserSpacesResponse{
		List: list,
	})
}

type CreateUserSpaceRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

type CreateUserSpaceResponse struct {
	SpaceID string `json:"space_id"`
}

func (s *HttpSrv) CreateUserSpace(c *gin.Context) {
	var (
		err error
		req CreateUserSpaceRequest
	)
	if err = utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	spaceID, err := v1.NewSpaceLogic(c, s.Core).CreateUserSpace(req.Title, req.Description)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, CreateUserSpaceResponse{
		SpaceID: spaceID,
	})
}

type ListSpaceUsersRequest struct {
	Page     uint64 `json:"page" binding:"required"`
	PageSize uint64 `json:"pagesize" binding:"required,lte=50"`
}

type ListSpaceUsersResponse struct {
	List  []types.User `json:"list"`
	Total int64        `json:"total"`
}

func (s *HttpSrv) ListSpaceUsers(c *gin.Context) {
	var (
		err error
		req ListSpaceUsersRequest
	)

	if err = utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	spaceID, _ := v1.InjectSpaceID(c)
	list, total, err := v1.NewSpaceLogic(c, s.Core).ListSpaceUsers(spaceID, req.Page, req.PageSize)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, ListSpaceUsersResponse{
		List:  list,
		Total: total,
	})
}

type SetUserSpaceRoleRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required"`
}

func (s *HttpSrv) SetUserSpaceRole(c *gin.Context) {
	var (
		err error
		req SetUserSpaceRoleRequest
	)
	if err = utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}
	spaceID, _ := v1.InjectSpaceID(c)
	if err = v1.NewSpaceLogic(c, s.Core).SetUserSpaceRole(spaceID, req.UserID, req.Role); err != nil {
		response.APIError(c, err)
		return
	}
	response.APISuccess(c, nil)
}

func (s *HttpSrv) UpdateSpace(c *gin.Context) {
	var (
		err error
		req CreateUserSpaceRequest
	)
	if err = utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}
	spaceID, _ := v1.InjectSpaceID(c)
	err = v1.NewSpaceLogic(c, s.Core).UpdateSpace(spaceID, req.Title, req.Description)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, nil)
}

func (s *HttpSrv) DeleteUserSpace(c *gin.Context) {
	spaceID, _ := v1.InjectSpaceID(c)
	err := v1.NewSpaceLogic(c, s.Core).DeleteUserSpace(spaceID)
	if err != nil {
		response.APIError(c, err)
		return
	}
	response.APISuccess(c, nil)
}

func (s *HttpSrv) LeaveSpace(c *gin.Context) {
	spaceID, _ := v1.InjectSpaceID(c)
	err := v1.NewSpaceLogic(c, s.Core).LeaveSpace(spaceID)
	if err != nil {
		response.APIError(c, err)
		return
	}
	response.APISuccess(c, nil)
}
