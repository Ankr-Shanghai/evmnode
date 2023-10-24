package public

type ErrJson struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

var (
	ErrNotExistCode = -32601
	ErrCode         = -32602
	WrongArgsErr    = "missing value for required argument"
	ErrNotExistMsg  = "the method does not exist/is not available"
)
