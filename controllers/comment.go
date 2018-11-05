package controllers

import (
	"net/http"
	"strconv"
	"fmt"
	"github.com/dchest/captcha" //验证码
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/Yq2/yqstudio/models"
	"github.com/Yq2/yqstudio/system"
)

//文章评论
func CommentPost(c *gin.Context) {
	s := sessions.Default(c)
	sessionUserID := s.Get(SESSION_KEY) //
	userId, _ := sessionUserID.(uint)
	//从post form表单拿到输入的验证码
	verifyCode := c.PostForm("verifyCode")
	captchaId := s.Get(SESSION_CAPTCHA) //从session里面拿到验证码id
	s.Delete(SESSION_CAPTCHA) //从session里面删除验证码
	_captchaId, _ := captchaId.(string) //验证码断言
	//比对验证码id和验证码是否匹配
	if !captcha.VerifyString(_captchaId, verifyCode) {
		//验证码ID和验证码数字不匹配，响应验证码错误
		//说明验证码数字 和  验证ID是有映射关系的
		c.JSON(http.StatusOK, gin.H {
			"succeed": false,
			"message": "验证码错误",
		})
		return
	}
	//验证码通过
	var err error
	postId := c.PostForm("postId") //form表单文章ID
	content := c.PostForm("content") //评论内容
	if len(content) == 0 {
		err = errors.New("评论不能为空.")
	}
	var post *models.Post
	post, err = models.GetPostById(postId) //根据文章ID获取文章
	if err == nil {
		pid, err := strconv.ParseUint(postId, 10, 64)
		if err == nil {
			//生成文章评论
			comment := &models.Comment {
				PostID:  uint(pid),
				Content: content, //评论内容
				UserID:  userId,
			}
			//保存文章评论
			err = comment.Insert()
		}
	}
	if err == nil {
		NotifyEmail("[YqStudio]您有一条新点评", fmt.Sprintf("<a href=\"%s/post/%d\" target=\"_blank\">%s</a>:%s", system.GetConfiguration().Domain, post.ID, post.Title, content))
		c.JSON(http.StatusOK, gin.H{
			"succeed": true,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"succeed": false,
			"message": err.Error(),
		})
	}
}

func CommentDelete(c *gin.Context) {
	s := sessions.Default(c)
	sessionUserID := s.Get(SESSION_KEY)
	userId, _ := sessionUserID.(uint)
	//获取restful url里面携带的参数
	commentId := c.Param("id")
	cid, err := strconv.ParseUint(commentId, 10, 64)
	if err == nil {
		comment := &models.Comment{
			UserID: uint(userId),
		}
		comment.ID = uint(cid)
		//删除用户ID名下ID 为comment.ID的评论
		err = comment.Delete()
	}
	c.JSON(http.StatusOK, gin.H{
		"succeed": err == nil,
	})
}

func CommentRead(c *gin.Context) {
	id := c.Param("id")
	_id, err := strconv.ParseUint(id, 10, 64)
	if err == nil {
		comment := new(models.Comment)
		comment.ID = uint(_id)
		err = comment.Update()
	}
	c.JSON(http.StatusOK, gin.H{
		"succeed": err == nil,
	})
}

func CommentReadAll(c *gin.Context) {
	err := models.SetAllCommentRead()
	c.JSON(http.StatusOK, gin.H{
		"succeed": err == nil,
	})
}
