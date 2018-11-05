package controllers

import (
	"net/http"
	"strings"
	"strconv"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/Yq2/yqstudio/models"
)

func SendMail(c *gin.Context) {
	subject := c.PostForm("subject")
	content := c.PostForm("content")
	userId := c.Query("userId")

	var err error
	if subject == "" || content == "" || userId == "" {
		err = errors.New("参数错误.")
	}
	if err == nil {
		var uid uint64
		uid, err = strconv.ParseUint(userId, 10, 64)
		if err == nil {
			var subscriber *models.Subscriber
			//根据ID获取订阅者
			subscriber, err = models.GetSubscriberById(uint(uid))
			if err == nil {
				err = sendMail(subscriber.Email, subject, content)
			}
		}
	}
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			"succeed": true,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"succeed": false,
			"msg":     err.Error(),
		})
	}
}

func SendBatchMail(c *gin.Context) {
	subject := c.PostForm("subject")
	content := c.PostForm("content")
	var err error
	if subject == "" || content == "" {
		err = errors.New("参数错误.")
	}
	if err == nil {
		var subscribers []*models.Subscriber
		subscribers, err = models.ListSubscriber(true)
		if err == nil {
			emails := make([]string, 0)
			for _, subscriber := range subscribers {
				emails = append(emails, subscriber.Email)
			}
			err = sendMail(strings.Join(emails, ";"), subject, content)
		}
	}
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			"succeed": true,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"succeed": false,
			"msg":     err.Error(),
		})
	}
}
