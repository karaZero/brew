package v1

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/starbx/brew-api/internal/core"
	"github.com/starbx/brew-api/pkg/ai"
	"github.com/starbx/brew-api/pkg/errors"
	"github.com/starbx/brew-api/pkg/i18n"
	"github.com/starbx/brew-api/pkg/types"
	"github.com/starbx/brew-api/pkg/types/protocol"
	"github.com/starbx/brew-api/pkg/utils"
)

type ReceiveFunc func(startAt int32, msg types.MessageContent, isIntercept bool) error
type DoneFunc func(startAt int32) error

// handleAssistantMessage 通过ws通知前端开始响应用户请求
func getReceiveFunc(ctx context.Context, core *core.Core, msg *types.ChatMessage) ReceiveFunc {
	imTopic := protocol.GenIMTopic(msg.SessionID)
	return func(startAt int32, message types.MessageContent, isIntercepted bool) error {
		if msg.Message == "" {
			msg.Message = string(message.Bytes())
		}

		completeStatus := types.MESSAGE_PROGRESS_GENERATING
		assistantStatus := types.WS_EVENT_ASSISTANT_CONTINUE
		if isIntercepted {
			completeStatus = types.MESSAGE_PROGRESS_INTERCEPTED
			assistantStatus = types.WS_EVENT_ASSISTANT_DONE
			// todo retry
			if err := core.Store().ChatMessageStore().RewriteMessage(ctx, msg.SpaceID, msg.SessionID, msg.ID, message.Bytes(), int32(completeStatus)); err != nil {
				slog.Error("failed to rewrite ai answer message to db", slog.String("session_id", msg.SessionID), slog.String("msg_id", msg.ID),
					slog.String("error", err.Error()))
				return err
			}
		} else {
			// todo retry
			if err := core.Store().ChatMessageStore().AppendMessage(ctx, msg.SpaceID, msg.SessionID, msg.ID, message.Bytes(), int32(completeStatus)); err != nil {
				slog.Error("failed to append ai answer message to db", slog.String("session_id", msg.SessionID), slog.String("msg_id", msg.ID),
					slog.String("error", err.Error()))
				return err
			}
		}

		if err := core.Srv().Tower().PublishStreamMessage(imTopic, assistantStatus, &types.StreamMessage{
			MessageID: msg.ID,
			SessionID: msg.SessionID,
			Message:   string(message.Bytes()),
			StartAt:   startAt,
			MsgType:   msg.MsgType,
			Complete:  int32(completeStatus),
		}); err != nil {
			slog.Error("failed to publish ai answer", slog.String("imtopic", imTopic), slog.String("error", err.Error()))
			return err
		}

		return nil
	}
}

// handleAssistantMessage 通过ws通知前端智能助理完成用户请求
func getDoneFunc(ctx context.Context, core *core.Core, msg *types.ChatMessage) DoneFunc {
	imTopic := protocol.GenIMTopic(msg.SessionID)
	return func(startAt int32) error {
		// todo retry
		assistantStatus := types.WS_EVENT_ASSISTANT_DONE
		completeStatus := types.MESSAGE_PROGRESS_COMPLETE
		message := ""
		if startAt == 0 {
			message = types.AssistantFailedMessage
			assistantStatus = types.WS_EVENT_ASSISTANT_FAILED
			completeStatus = types.MESSAGE_PROGRESS_FAILED
			slog.Error("assistant response is empty, will delete assistant answer message", slog.String("session_id", msg.SessionID), slog.String("msg_id", msg.ID))
			// 返回了0个字符就完成的情况一般是assistant服务异常，无响应，服务端删除该消息，避免数据库存在空记录
			if err := core.Store().ChatMessageStore().DeleteMessage(ctx, msg.ID); err != nil {
				slog.Error("failed to delete assistant answer message", slog.String("session_id", msg.SessionID), slog.String("msg_id", msg.ID),
					slog.String("error", err.Error()))
				return err
			}
		} else {
			if err := core.Store().ChatMessageStore().UpdateMessageCompleteStatus(ctx, msg.SessionID, msg.ID, int32(types.MESSAGE_PROGRESS_COMPLETE)); err != nil {
				slog.Error("failed to finished assistant answer message", slog.String("session_id", msg.SessionID), slog.String("msg_id", msg.ID),
					slog.String("error", err.Error()))
				return err
			}
		}

		if err := core.Srv().Tower().PublishStreamMessage(imTopic, assistantStatus, &types.StreamMessage{
			MessageID: msg.ID,
			SessionID: msg.SessionID,
			Complete:  int32(completeStatus),
			MsgType:   msg.MsgType,
			Message:   message,
			StartAt:   startAt,
		}); err != nil {
			slog.Error("failed to publish gpt answer", slog.String("imtopic", imTopic), slog.String("error", err.Error()))
			return err
		}
		return nil
	}
}

func notifyAssistantMessageInitialized(core *core.Core, msg *types.ChatMessage) error {
	imTopic := protocol.GenIMTopic(msg.SessionID)
	if err := core.Srv().Tower().PublishMessageMeta(imTopic, types.WS_EVENT_ASSISTANT_INIT, chatMsgToTextMsg(msg)); err != nil {
		slog.Error("failed to publish ai message builded event", slog.String("imtopic", imTopic))
		return err
	}
	return nil
}

func handleAndNotifyAssistantFailed(core *core.Core, aiMessage *types.ChatMessage, err error) error {
	imTopic := protocol.GenIMTopic(aiMessage.SessionID)
	content := types.AssistantFailedMessage
	completeStatus := types.MESSAGE_PROGRESS_FAILED
	if err == context.Canceled { // 用户手动终止 会关闭上下文
		completeStatus = types.MESSAGE_PROGRESS_CANCELED
		content = ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := core.Store().ChatMessageStore().UpdateMessageCompleteStatus(ctx, aiMessage.SessionID, aiMessage.ID, int32(completeStatus)); err != nil {
		slog.Error("failed to finished ai answer message", slog.String("session_id", aiMessage.SessionID), slog.String("msg_id", aiMessage.ID),
			slog.String("error", err.Error()))
	}

	if err := core.Srv().Tower().PublishStreamMessage(imTopic, types.WS_EVENT_ASSISTANT_FAILED, &types.StreamMessage{
		MessageID: aiMessage.ID,
		SessionID: aiMessage.SessionID,
		Complete:  int32(completeStatus),
		MsgType:   aiMessage.MsgType,
		Message:   content,
	}); err != nil {
		slog.Error("failed to publish gpt answer", slog.String("imtopic", imTopic), slog.String("error", err.Error()))
		return err
	}
	return nil
}

// requestAI
func requestAI(ctx context.Context, core *core.Core, sessionContext *SessionContext, docs *types.RAGDocs, receiveFunc ReceiveFunc, done DoneFunc) error {
	slog.Debug("request to ai", slog.Any("context", sessionContext.MessageContext), slog.String("prompt", sessionContext.Prompt))

	requestCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	tool := core.Srv().AI().NewQuery(requestCtx, sessionContext.MessageContext)
	tool.WithPrompt(sessionContext.Prompt)

	resp, err := tool.QueryStream()
	if err != nil {
		return err
	}

	marks := make(map[string]string)

	for _, v := range docs.Docs {
		for fake, real := range v.SW.Map() {
			marks[fake] = real
		}
	}

	respChan, err := ai.HandleAIStream(requestCtx, resp, marks)
	if err != nil {
		return errors.New("requestAI.HandleAIStream", i18n.ERROR_INTERNAL, err)
	}

	// 3. handle response
	var sended []rune
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-respChan:
			// slog.Debug("got ai response", slog.Any("msg", msg), slog.Bool("status", ok))
			if !ok {
				done(int32(len(sended)))
				return nil
			}
			if msg.Error != nil {
				return err
			}
			if msg.FinishReason != "" && msg.FinishReason != "stop" {
				slog.Error("AI srv unexpected exit", slog.String("error", msg.FinishReason), slog.String("id", msg.ID))
				return errors.New("requestAI.Srv.AI.Query", i18n.ERROR_INTERNAL, fmt.Errorf("%s", msg.FinishReason))
			}

			if msg.Message != "" {
				if err := receiveFunc(int32(len(sended)), &types.TextMessage{Text: msg.Message}, false); err != nil {
					return errors.New("ChatGPTLogic.RequestChatGPT.for.respChan.receive", i18n.ERROR_INTERNAL, err)
				}
				sended = append(sended, []rune(msg.Message)...)
			}
		}
	}
}

func NewNormalAssistant(core *core.Core) *NormalAssistant {
	return &NormalAssistant{
		core: core,
	}
}

type NormalAssistant struct {
	core *core.Core
}

func (s *NormalAssistant) InitAssistantMessage(ctx context.Context, userReqMessage *types.ChatMessage, ext types.ChatMessageExt) (*types.ChatMessage, error) {
	// 生成ai响应消息载体的同时，写入关联的内容列表(ext)
	return initAssistantMessage(ctx, s.core, userReqMessage, ext)
}

// GenSessionContext 生成session上下文
func (s *NormalAssistant) GenSessionContext(ctx context.Context, prompt string, reqMsgWithDocs *types.ChatMessage) (*SessionContext, error) {
	// latency := s.core.Metrics().GenContextTimer("GenChatSessionContext")
	// defer latency.ObserveDuration()
	return GenChatSessionContextAndSummaryIfExceedsTokenLimit(ctx, s.core, prompt, reqMsgWithDocs, normalGenMessageCondition, types.GEN_CONTEXT)
}

// RequestAssistant 向智能助理发起请求
// reqMsgInfo 用户请求的内容
// recvMsgInfo 用于承载ai回复的内容，会预先在数据库中为ai响应的数据创建出对应的记录
func (s *NormalAssistant) RequestAssistant(ctx context.Context, docs *types.RAGDocs, reqMsgWithDocs *types.ChatMessage, recvMsgInfo *types.ChatMessage) error {
	prompt := ai.BuildRAGPrompt(s.core.Cfg().Prompt.Query, ai.NewDocs(docs.Docs), s.core.Srv().AI())
	chatSessionContext, err := s.GenSessionContext(ctx, prompt, reqMsgWithDocs)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*3)
	defer cancel()
	receiveFunc := getReceiveFunc(ctx, s.core, recvMsgInfo)
	doneFunc := getDoneFunc(ctx, s.core, recvMsgInfo)
	if err = requestAI(ctx, s.core, chatSessionContext, docs, receiveFunc, doneFunc); err != nil {
		slog.Error("failed to request AI", slog.String("error", err.Error()))
		return handleAndNotifyAssistantFailed(s.core, recvMsgInfo, err)
	}
	return nil
}

func initAssistantMessage(ctx context.Context, core *core.Core, userReqMsg *types.ChatMessage, ext types.ChatMessageExt) (*types.ChatMessage, error) {
	answerMsg, err := prepareTheAnswerMsg(ctx, core, userReqMsg.SpaceID, userReqMsg.SessionID)
	if err != nil {
		slog.Error("failed to generate answer message for ai", slog.String("session_id", userReqMsg.SessionID), slog.String("error", err.Error()))
		return nil, err
	}

	answerMsg.MsgBlock = userReqMsg.MsgBlock
	answerMsg.UserID = userReqMsg.UserID // ai answer message is also belong to user

	err = core.Store().Transaction(ctx, func(ctx context.Context) error {
		if err = core.Store().ChatMessageStore().Create(ctx, answerMsg); err != nil {
			slog.Error("failed to insert ai answer message to db", slog.String("msg_id", answerMsg.ID), slog.String("session_id", answerMsg.SessionID), slog.String("error", err.Error()))
			return err
		}

		ext.MessageID = answerMsg.ID

		if err = core.Store().ChatMessageExtStore().Create(ctx, ext); err != nil {
			slog.Error("failed to insert ai answer ext to db", slog.String("msg_id", answerMsg.ID), slog.String("session_id", answerMsg.SessionID), slog.String("error", err.Error()))
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return answerMsg, nil
}

func prepareTheAnswerMsg(ctx context.Context, core *core.Core, spaceID, sessionID string) (*types.ChatMessage, error) {
	// generate message meta
	msgID := core.Srv().SeqSrv().GenMessageID()
	seqID, err := core.Srv().SeqSrv().GetChatSessionSeqID(ctx, spaceID, sessionID)
	if err != nil {
		slog.Error("Failed to get session message sequence id", slog.String("session_id", sessionID), slog.String("error", err.Error()))
		return nil, err
	}
	// pre-generate response messages
	return genUncompleteAIMessage(spaceID, sessionID, msgID, seqID), nil
}

// generate uncomplete ai response message meta
func genUncompleteAIMessage(spaceID, sessionID, msgID string, seqID int64) *types.ChatMessage {
	return &types.ChatMessage{
		ID:        msgID,
		SpaceID:   spaceID,
		Sequence:  seqID,
		Role:      types.USER_ROLE_ASSISTANT,
		SendTime:  time.Now().Unix(),
		MsgType:   types.MESSAGE_TYPE_TEXT,
		Complete:  types.MESSAGE_PROGRESS_UNCOMPLETE,
		SessionID: sessionID,
	}
}

type messageCondition func(historyMsgID, inputMsgID string) bool

func normalGenMessageCondition(historyMsgID, inputMsgID string) bool {
	return historyMsgID > inputMsgID
}

func reGenMessageCondition(historyMsgID, inputMsgID string) bool {
	return historyMsgID >= inputMsgID
}

func appendSummaryToPromptMsg(msg *types.MessageContext, summary *types.ChatSummary) {
	// Sprintf 是个比较低效的字符串拼接方法，当前量级可以暂且这么做，量级上来以后可以优化到 strings.Builder
	msg.Content = fmt.Sprintf("%s, You will continue the conversation with understanding the context. The following is the context for conversation：{ %s }", msg.Content, summary.Content)
}

func isErrorMessage(msg string) bool {
	msg = strings.TrimSpace(msg)
	if strings.HasPrefix(msg, "Sorry，") || strings.HasPrefix(msg, "抱歉，") || msg == "" {
		return true
	}
	return false
}

// genChatSessionContextAndSummaryIfExceedsTokenLimit 生成gpt请求上下文
func GenChatSessionContextAndSummaryIfExceedsTokenLimit(ctx context.Context, core *core.Core, basePrompt string, reqMsgWithDocs *types.ChatMessage, msgCondition messageCondition, justGenSummary types.SystemContextGenConditionType) (*SessionContext, error) {
	reGen := false

ReGen:
	var reqMsg []*types.MessageContext
	summary, err := core.Store().ChatSummaryStore().GetChatSessionLatestSummary(ctx, reqMsgWithDocs.SessionID)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("genDialogContextAndSummaryIfExceedsTokenLimit.ChatSummaryStore.GetChatSessionLatestSummary", i18n.ERROR_INTERNAL, err)
	}

	if basePrompt != "" {
		reqMsg = append(reqMsg, &types.MessageContext{
			Role:    types.USER_ROLE_SYSTEM,
			Content: basePrompt,
		})

		if summary != nil {
			appendSummaryToPromptMsg(reqMsg[0], summary)
		}
	}

	if summary == nil {
		summary = &types.ChatSummary{}
	}

	// 获取比summary msgid更大的聊天内容组成上下文
	msgList, err := core.Store().ChatMessageStore().ListSessionMessage(ctx, reqMsgWithDocs.SpaceID, reqMsgWithDocs.SessionID, summary.MessageID, types.NO_PAGING, types.NO_PAGING)
	if err != nil {
		return nil, errors.New("genDialogContextAndSummaryIfExceedsTokenLimit.ChatMessageStore.ListSessionMessage", i18n.ERROR_INTERNAL, err)
	}

	// 对消息按msgid进行排序
	sort.Slice(msgList, func(i, j int) bool {
		return msgList[i].ID < msgList[j].ID
	})

	var (
		summaryMessageCutRange int
		summaryMessageID       string
		contextIndex           int
	)

	for _, v := range msgList {
		if isErrorMessage(v.Message) {
			continue
		}

		if v.Complete != types.MESSAGE_PROGRESS_COMPLETE {
			continue
		}

		if msgCondition(v.ID, reqMsgWithDocs.ID) {
			// 当前逻辑回复的是 msgID, 所以上下文中不应该出现晚于 msgID 出现的消息，多人场景会有此情况
			break
		}

		contextIndex++
		message := v.Message
		if v.ID == reqMsgWithDocs.ID {
			message = reqMsgWithDocs.Message
		}

		reqMsg = append(reqMsg, &types.MessageContext{
			Role:    v.Role,
			Content: message,
		})
	}

	if contextIndex > 0 {
		if contextIndex >= 3 { // 如果聊天记录追加超过3条，则在总结前保留最新的三条消息，否则保留最后一条
			summaryMessageCutRange = 3
		} else {
			summaryMessageCutRange = 1
		}
		summaryMessageID = msgList[contextIndex-summaryMessageCutRange].ID
	}

	// 计算token是否超出限额，超出20条记录自动做一次总结
	if len(msgList) > 20 || core.Srv().AI().MsgIsOverLimit(reqMsg) {
		if len(reqMsg) <= 3 || reGen {
			// 表明当前prompt + 总结 + 用户一段对话已经超出 max token
			slog.Warn("the current context token is insufficient", slog.String("session_id", reqMsgWithDocs.SessionID), slog.String("msg_id", reqMsgWithDocs.ID))
			return nil, errors.New("genDialogContextAndSummaryIfExceedsTokenLimit.MessageStore.ListDialogMessage", "the current dialog token is insufficient", err)
		}

		summaryReq := reqMsg[:len(reqMsg)-summaryMessageCutRange]
		if core.Srv().AI().MsgIsOverLimit(summaryReq) {
			// 历史数据迁移可能导致某些用户的历史聊天记录过大，无法生成总结，若超出limit，则每次删除第一条消息(prompt后的第一条消息，故索引为1)
			for {
				summaryReq = lo.Drop(summaryReq, 1)
				if !core.Srv().AI().MsgIsOverLimit(summaryReq) {
					break
				}
			}
		}

		reGen = true
		// 生成新的总结
		if err = genChatSessionContextSummary(ctx, core, reqMsgWithDocs.SessionID, summaryMessageID, summaryReq); err != nil {
			return nil, errors.Trace("genDialogContextAndSummaryIfExceedsTokenLimit.genDialogContextSummary", err)
		}
		if justGenSummary == types.GEN_SUMMARY_ONLY {
			return nil, nil
		}
		goto ReGen
	}
	return &SessionContext{
		Prompt:         basePrompt,
		MessageID:      reqMsgWithDocs.ID,
		SessionID:      reqMsgWithDocs.SessionID,
		MessageContext: reqMsg,
	}, nil
}

type SessionContext struct {
	MessageID      string
	SessionID      string
	MessageContext []*types.MessageContext
	Prompt         string
}

// genChatSessionContextSummary 生成dialog上下文总结
func genChatSessionContextSummary(ctx context.Context, core *core.Core, sessionID, summaryMessageID string, reqMsg []*types.MessageContext) error {
	slog.Debug("start generating context summary", slog.String("session_id", sessionID), slog.String("msg_id", summaryMessageID), slog.Any("request_message", reqMsg))
	prompt := core.Cfg().Prompt.ChatSummary
	if prompt == "" {
		prompt = ai.PROMPT_SUMMARY_DEFAULT_EN
	}

	queryOpts := core.Srv().AI().NewQuery(ctx, reqMsg)
	queryOpts.WithPrompt(prompt)

	// 总结仍然使用v3来生成
	resp, err := queryOpts.Query()
	if err != nil || len(resp.Received) == 0 {
		slog.Error("failed to generate dialog context summary", slog.String("error", err.Error()), slog.Any("response", resp))
		return errors.New("genDialogContextSummary.gptSrv.Chat", i18n.ERROR_INTERNAL, err)
	}

	if len(resp.Received) > 1 {
		slog.Warn("chat method response multi line content", slog.Any("response", resp))
	}

	if err = core.Store().ChatSummaryStore().Create(ctx, types.ChatSummary{
		ID:        utils.GenSpecIDStr(),
		SessionID: sessionID,
		MessageID: summaryMessageID,
		Content:   resp.Received[0],
	}); err != nil {
		return errors.New("genDialogContextSummary.ChatSummaryStore.Create", i18n.ERROR_INTERNAL, err)
	}
	slog.Debug("succeed to generate summary", slog.String("session_id", sessionID), slog.String("msg_id", summaryMessageID))
	return nil
}
