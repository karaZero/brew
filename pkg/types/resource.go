package types

type Resource struct {
	ID          string `json:"id" db:"id"`                     // 资源的唯一标识
	Title       string `json:"title" db:"title"`               // 资源标题
	UserID      string `json:"user_id" db:"user_id"`           // 用户id
	SpaceID     string `json:"space_id" db:"space_id"`         // 资源所属空间ID
	Description string `json:"description" db:"description"`   // 资源描述信息
	Cycle       int    `json:"cycle" db:"cycle"`               // 资源周期，0为不限制
	Prompt      string `json:"prompt" db:"prompt"`             // 自定义prompt
	CreatedTime int64  `json:"created_time" db:"created_time"` // 资源创建时间，UNIX时间戳
}