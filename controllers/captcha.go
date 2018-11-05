package controllers

import (
	"github.com/dchest/captcha" //验证码库
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func CaptchaGet(c *gin.Context) {
	//获取session
	session := sessions.Default(c)
	captchaId := captcha.NewLen(4) //生成一个4位长的验证码
	session.Delete(SESSION_CAPTCHA) //删除GIN_CAPTCHA验证码字段
	session.Set(SESSION_CAPTCHA, captchaId) //重新设置验证码
	session.Save()
	//向响应体里面写入长100，宽40的图片验证码
	captcha.WriteImage(c.Writer, captchaId, 100, 40)
}
