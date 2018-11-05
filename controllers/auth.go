package controllers

import (
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/Yq2/yqstudio/helpers"
	"github.com/Yq2/yqstudio/system"
	"net/http"
)

func AuthGet(c *gin.Context) {
	//获取 restful 参数authType
	authType := c.Param("authType")
	//获取连接中的session
	session := sessions.Default(c)
	uuid := helpers.UUID() //生成一个UUID
	session.Delete(SESSION_GITHUB_STATE) //删除 GITHUB_STATE
	session.Set(SESSION_GITHUB_STATE, uuid) //将GITHUB_STATE 重新设置为当前UUID
	session.Save() //保存session

	authurl := "/signin"
	switch authType {
	case "github":
		authurl = fmt.Sprintf(system.GetConfiguration().GithubAuthUrl, system.GetConfiguration().GithubClientId, uuid)
	case "weibo": //待处理
	case "qq": //待处理
	case "wechat": //待处理
	case "oschina": //待处理
	default:
	}
	//302重定向
	c.Redirect(http.StatusFound, authurl)
}
