package api

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorData  `json:"error,omitempty"`
}

type ErrorData struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func SuccessResponse(data interface{}) Response {
	return Response{
		Success: true,
		Data:    data,
	}
}

func ErrorResponse(message, code string) Response {
	return Response{
		Success: false,
		Error: &ErrorData{
			Message: message,
			Code:    code,
		},
	}
}