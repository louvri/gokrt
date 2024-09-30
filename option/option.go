package option

type Option string

var RUN_WITH_ERROR Option = "RUN ON ERROR"
var RUN_ASYNC Option = "RUN ASYNC"
var EXECUTE_AFTER Option = "EXECUTE AFTER"
var EXECUTE_BEFORE Option = "EXECUTE BEFORE"
var FORBID_CURRENT_ENDPOINT_RUN Option = "FORBID CURRENT ENDPOINT RUN"
var RUN_IN_TRANSACTION Option = "RUN IN TRANSACTION"

type Config struct {
	CacheKey string
	Option   []Option
}
