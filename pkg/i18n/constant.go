package i18n

var ALLOW_LANG = map[string]bool{
	"en":    true,
	"zh-CN": true,
}

const DEFAULT_LANG = "en"

const (
	ERROR_INTERNAL          = "error.internal"
	ERROR_NOTFOUND          = "error.notfound"
	ERROR_INVALIDARGUMENT   = "error.invalidargument"
	ERROR_PERMISSION_DENIED = "error.permission.denied"
	ERROR_UNAUTHORIZED      = "error.unauthorized"
	ERROR_EXIST             = "error.exist"
	ERROR_FORBIDDEN         = "error.forbidden"
	ERROR_TOO_MANY_REQUESTS = "error.tooManyRequests"

	ERROR_INVALID_TOKEN   = "error.invalid.token"
	ERROR_INVALID_ACCOUNT = "error.invalid.account"

	ERROR_LOGIC_VECTOR_DB_NOT_MATCHED_CONTENT_DB = "error.logic.vector.db.notmatch.content.db"
)
