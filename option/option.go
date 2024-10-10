package option

type Option string

var RUN_WITH_ERROR Option = "RUN ON ERROR"
var RUN_ASYNC_WAIT Option = "RUN ASYNC WAIT"
var EXECUTE_AFTER Option = "EXECUTE AFTER"
var EXECUTE_BEFORE Option = "EXECUTE BEFORE"
var RUN_IN_TRANSACTION Option = "RUN IN TRANSACTION"

type Config struct {
	CacheKey string
	Option   []Option
}
