package service

import (
	"github.com/gin-gonic/gin"

	"github.com/starbx/brew-api/cmd/service/handler"
	"github.com/starbx/brew-api/internal/core"
	"github.com/starbx/brew-api/internal/core/srv"
	v1 "github.com/starbx/brew-api/internal/logic/v1"
	"github.com/starbx/brew-api/internal/response"
)

func serve(core *core.Core) {
	engine := gin.New()
	httpSrv := &handler.HttpSrv{
		Core:   core,
		Engine: engine,
	}
	setupHttpRouter(httpSrv)

	engine.Run(core.Cfg().Addr)
}

func getUserLimitBuilder(core *core.Core) func(key string) gin.HandlerFunc {
	return func(key string) gin.HandlerFunc {
		return UseLimit(core, key, func(c *gin.Context) string {
			token, _ := v1.InjectTokenClaim(c)
			return key + ":" + token.User
		})
	}
}

func getSpaceLimitBuilder(core *core.Core) func(key string) gin.HandlerFunc {
	return func(key string) gin.HandlerFunc {
		return UseLimit(core, key, func(c *gin.Context) string {
			spaceid, _ := c.Params.Get("spaceid")
			return key + ":" + spaceid
		})
	}
}

func setupHttpRouter(s *handler.HttpSrv) {
	userLimit := getUserLimitBuilder(s.Core)
	spaceLimit := getSpaceLimitBuilder(s.Core)
	// auth
	s.Engine.Use(I18n(), response.NewResponse())
	s.Engine.Use(Cors)
	apiV1 := s.Engine.Group("/api/v1")
	{
		apiV1.GET("/connect", AuthorizationFromQuery(s.Core), handler.Websocket(s.Core))
		apiV1.POST("/login/token", Authorization(s.Core), s.AccessLogin)
		authed := apiV1.Group("")
		authed.Use(Authorization(s.Core))
		user := authed.Group("/user")
		{
			user.PUT("/profile", s.UpdateUserProfile)
		}

		space := authed.Group("/space")
		{
			space.GET("/list", s.ListUserSpaces)
			space.DELETE("/:spaceid/leave", VerifySpaceIDPermission(s.Core, srv.PermissionView), s.LeaveSpace)

			space.POST("", userLimit("modify_space"), s.CreateUserSpace)

			space.Use(VerifySpaceIDPermission(s.Core, srv.PermissionAdmin))
			space.DELETE("/:spaceid", s.DeleteUserSpace)
			space.PUT("/:spaceid", s.UpdateSpace)
			space.PUT("/:spaceid/user/role", userLimit("modify_space"), s.SetUserSpaceRole)
			space.GET("/:spaceid/users", s.ListSpaceUsers)
		}

		knowledge := authed.Group("/:spaceid/knowledge")
		{
			viewScope := knowledge.Group("")
			{
				viewScope.Use(VerifySpaceIDPermission(s.Core, srv.PermissionView))
				viewScope.GET("", s.GetKnowledge)
				viewScope.GET("/list", spaceLimit("knowledge_list"), s.ListKnowledge)
				viewScope.POST("/query", spaceLimit("query"), s.Query)
			}

			editScope := knowledge.Group("")
			{
				editScope.Use(VerifySpaceIDPermission(s.Core, srv.PermissionEdit), spaceLimit("knowledge_modify"))
				editScope.POST("", s.CreateKnowledge)
				editScope.PUT("", s.UpdateKnowledge)
				editScope.DELETE("", s.DeleteKnowledge)
			}
		}

		resource := authed.Group("/:spaceid/resource")
		{
			resource.Use(VerifySpaceIDPermission(s.Core, srv.PermissionView))
			resource.GET("", s.GetResource)
			resource.GET("/list", s.ListResource)

			resource.Use(spaceLimit("resource"))
			resource.POST("", s.CreateResource)
			resource.PUT("", s.UpdateResource)
			resource.DELETE("/:resourceid", s.DeleteResource)
		}

		chat := authed.Group("/:spaceid/chat")
		{
			chat.Use(VerifySpaceIDPermission(s.Core, srv.PermissionView))
			chat.POST("", s.CreateChatSession)
			chat.DELETE("/:session", s.DeleteChatSession)
			chat.GET("/list", s.ListChatSession)
			chat.POST("/:session/message/id", s.GenMessageID)
			chat.PUT("/:session/named", spaceLimit("named_session"), s.RenameChatSession)
			chat.GET("/:session/message/:messageid/ext", s.GetChatMessageExt)

			history := chat.Group("/:session/history")
			{
				history.GET("/list", s.GetChatSessionHistory)
			}

			message := chat.Group("/:session/message")
			{
				message.Use(spaceLimit("create_message"))
				message.POST("", s.CreateChatMessage)
			}
		}
	}
}
