package types

import "fmt"

type TableName string

func (s TableName) Name() string {
	return fmt.Sprintf("%s%s", TABLE_PREFIX, s)
}

const TABLE_PREFIX = "bw_"

const (
	TABLE_KNOWLEDGE        = TableName("knowledge")
	TABLE_KNOWLEDGE_CHUNK  = TableName("knowledge_chunk")
	TABLE_VECTORS          = TableName("vectors")
	TABLE_ACCESS_TOKEN     = TableName("access_token")
	TABLE_USER_SPACE       = TableName("user_space")
	TABLE_SPACE            = TableName("space")
	TABLE_RESOURCE         = TableName("resource")
	TABLE_USER             = TableName("user")
	TABLE_CHAT_SESSION     = TableName("chat_session")
	TABLE_CHAT_MESSAGE     = TableName("chat_message")
	TABLE_CHAT_SUMMARY     = TableName("chat_summary")
	TABLE_CHAT_MESSAGE_EXT = TableName("chat_message_ext")
)
