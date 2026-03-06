package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一结构体封装
type Response struct {
	Code int         `json:"code"` // 业务状态码
	Msg  string      `json:"msg"`  // 提示信息
	Data interface{} `json:"data"` // 数据载荷
}

// 常见业务状态码定义
const (
	CodeSuccess = 0
	CodeParam   = 10001
	CodeAuth    = 30001
	CodeServer  = 50000
)

// Result 是最底层的输出方法
// 真正的业务逻辑成败是由 JSON 体里的 Code 决定的，这是业界的标准做法。
func Result(c *gin.Context, code int, msg string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}

// Success 成功响应带数据
func Success(c *gin.Context, data interface{}) {
	Result(c, CodeSuccess, "success", data)
}

// SuccessWithMessage 成功响应带自定义消息
func SuccessWithMessage(c *gin.Context, msg string, data interface{}) {
	Result(c, CodeSuccess, msg, data)
}

// Fail 失败响应 (自定义业务码和消息)
func Fail(c *gin.Context, code int, msg string) {
	Result(c, code, msg, nil)
}

// FailWithParam 参数错误快捷响应
func FailWithParam(c *gin.Context, msg string) {
	Result(c, CodeParam, msg, nil)
}

// FailWithServer 服务器内部错误快捷响应
func FailWithServer(c *gin.Context, msg string) {
	Result(c, CodeServer, msg, nil)
}
