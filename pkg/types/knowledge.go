package types

import (
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
)

const (
	DEFAULT_RESOURCE = "knowledge"
)

// export const cards = pgTable('cards', {
//   id: uuid('id').primaryKey().notNull().defaultRandom(),
//   spaceID: uuid('spaceID')
//     .notNull()
//     .references(() => spaces.id),
//   kind: cardKindEnum('kind').notNull(),
//   // tags is a string slice
//   tags: jsonb('tags').default([]),
//   content: text('content'),
//   authorID: varchar('authorID', { length: 256 }).notNull(),
//   createdAt: timestamp('createdAt', {
//     mode: 'string',
//     withTimezone: true,
//   }).defaultNow(),
//   updatedAt: timestamp('updatedAt', {
//     mode: 'string',
//     withTimezone: true,
//   }).defaultNow(),
// })

type KnowledgeKind string

const (
	KNOWLEDGE_KIND_TEXT    KnowledgeKind = "text"
	KNOWLEDGE_KIND_IMAGE                 = "image"
	KNOWLEDGE_KIND_VIDEO                 = "video"
	KNOWLEDGE_KIND_URL                   = "url"
	KNOWLEDGE_KIND_UNKNOWN               = "unknown"
)

func KindNewFromString(s string) KnowledgeKind {
	switch strings.ToLower(s) {
	case string(KNOWLEDGE_KIND_TEXT):
		return KNOWLEDGE_KIND_TEXT
	case string(KNOWLEDGE_KIND_IMAGE):
		return KNOWLEDGE_KIND_IMAGE
	case string(KNOWLEDGE_KIND_VIDEO):
		return KNOWLEDGE_KIND_VIDEO
	default:
		return KNOWLEDGE_KIND_UNKNOWN
	}
}

func (k KnowledgeKind) String() string {
	return string(k)
}

type KnowledgeStage int8

const (
	KNOWLEDGE_STAGE_NONE      KnowledgeStage = 0
	KNOWLEDGE_STAGE_SUMMARIZE KnowledgeStage = 1
	KNOWLEDGE_STAGE_EMBEDDING KnowledgeStage = 2
	KNOWLEDGE_STAGE_DONE      KnowledgeStage = 3
)

var namesForKnowledgeStage = map[KnowledgeStage]string{
	KNOWLEDGE_STAGE_NONE:      "None",
	KNOWLEDGE_STAGE_SUMMARIZE: "Summarize",
	KNOWLEDGE_STAGE_EMBEDDING: "Embedding",
	KNOWLEDGE_STAGE_DONE:      "Done",
}

func (v KnowledgeStage) String() string {
	if n, ok := namesForKnowledgeStage[v]; ok {
		return n
	}
	return fmt.Sprintf("KnowledgeStage(%d)", v)
}

func (v KnowledgeStage) int8() int8 {
	return int8(v)
}

type KnowledgeLite struct {
	ID       string         `json:"id" db:"id"`
	SpaceID  string         `json:"space_id" db:"space_id"`
	Resource string         `json:"resource" db:"resource"`
	Title    string         `json:"title" db:"title"`
	Tags     pq.StringArray `json:"tags" db:"tags"`
	UserID   string         `json:"user_id" db:"user_id"`
}

type Knowledge struct {
	ID         string         `json:"id" db:"id"`
	SpaceID    string         `json:"space_id" db:"space_id"`
	Kind       KnowledgeKind  `json:"kind" db:"kind"`
	Resource   string         `json:"resource" db:"resource"`
	Title      string         `json:"title" db:"title"`
	Tags       pq.StringArray `json:"tags" db:"tags"`
	Content    string         `json:"content" db:"content"`
	UserID     string         `json:"user_id" db:"user_id"`
	Summary    string         `json:"summary" db:"summary"`
	MaybeDate  string         `json:"maybe_date" db:"maybe_date"`
	Stage      KnowledgeStage `json:"stage" db:"stage"`
	CreatedAt  int64          `json:"created_at" db:"created_at"`
	UpdatedAt  int64          `json:"updated_at" db:"updated_at"`
	RetryTimes int            `json:"retry_times" db:"retry_times"`
}

type GetKnowledgeOptions struct {
	ID         string
	IDs        []string
	Kind       []KnowledgeKind
	SpaceID    string
	UserID     string
	Resource   *ResourceQuery
	Stage      KnowledgeStage
	RetryTimes int
}

func (opts GetKnowledgeOptions) Apply(query *sq.SelectBuilder) {
	if opts.ID != "" {
		*query = query.Where(sq.Eq{"id": opts.ID})
	} else if len(opts.IDs) > 0 {
		*query = query.Where(sq.Eq{"id": opts.IDs})
	}
	if opts.SpaceID != "" {
		*query = query.Where(sq.Eq{"space_id": opts.SpaceID})
	}
	if opts.UserID != "" {
		*query = query.Where(sq.Eq{"user_id": opts.UserID})
	}
	if opts.Resource != nil {
		*query = query.Where(opts.Resource.ToQuery())
	}
	if len(opts.Kind) > 0 {
		*query = query.Where(sq.Eq{"kind": opts.Kind})
	}
	if opts.Stage > 0 {
		*query = query.Where(sq.Eq{"stage": opts.Stage})
	}
	if opts.RetryTimes > 0 {
		*query = query.Where(sq.Eq{"retry_times": opts.RetryTimes})
	}
}

type ResourceQuery struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

func (r *ResourceQuery) ToQuery() sq.Sqlizer {
	if len(r.Include) > 0 {
		return sq.Eq{"resource": r.Include}
	}
	return sq.NotEq{"resource": r.Exclude}
}

type UpdateKnowledgeArgs struct {
	Title    string
	Resource string
	Kind     KnowledgeKind
	Content  string
	Tags     []string
	Stage    KnowledgeStage
	Summary  string
}
