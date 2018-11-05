package controllers

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/Yq2/yqstudio/helpers"
	"github.com/Yq2/yqstudio/models"
	"github.com/Yq2/yqstudio/system"
)

func SubscribeGet(c *gin.Context) {
	count, _ := models.CountSubscriber()
	c.HTML(http.StatusOK, "other/subscribe.html", gin.H{
		"total": count,
	})
}

func Subscribe(c *gin.Context) {
	//从form表单拿到mail参数
	mail := c.PostForm("mail")
	var err error
	if len(mail) > 0 {
		var subscriber *models.Subscriber //订阅者模型
		subscriber, err = models.GetSubscriberByEmail(mail) //查询mail的订阅
		if err == nil {
			//订阅者验证状态为0 且过期时间超过当前时间
			if !subscriber.VerifyState && helpers.GetCurrentTime().After(subscriber.OutTime) { //激活链接超时
				err = sendActiveEmail(subscriber) //邮箱验证
				if err == nil {
					count, _ := models.CountSubscriber() //统计所有有效订阅者
					c.HTML(http.StatusOK, "other/subscribe.html", gin.H{
						"message": "订阅成功.",
						"total":   count,
					})
					return
				}
			} else if subscriber.VerifyState && !subscriber.SubscribeState { //已认证，未订阅
				subscriber.SubscribeState = true //将订阅状态设置为true
				err = subscriber.Update() //更新订阅状态
				if err == nil {
					err = errors.New("订阅成功.")
				}
			} else {
				err = errors.New("邮箱没有激活，请登录你绑定的邮箱进行验证.")
			}
		} else {
			subscriber := &models.Subscriber{
				Email: mail,
				OutTime:helpers.GetCurrentTime(),
			}
			err = subscriber.Insert()
			if err == nil {
				err = sendActiveEmail(subscriber)
				if err == nil {
					count, _ := models.CountSubscriber()
					c.HTML(http.StatusOK, "other/subscribe.html", gin.H{
						"message": "订阅成功.",
						"total":   count,
					})
					return
				}
			}
		}
	} else {
		err = errors.New("邮箱地址不能为空.")
	}
	count, _ := models.CountSubscriber()
	c.HTML(http.StatusOK, "other/subscribe.html", gin.H {
		"message": err.Error(),
		"total":   count,
	})
}

//邮箱验证
func sendActiveEmail(subscriber *models.Subscriber) error {
	uuid := helpers.UUID()
	duration, _ := time.ParseDuration("30m")
	subscriber.OutTime = helpers.GetCurrentTime().Add(duration) //新的过期时间
	subscriber.SecretKey = uuid //uuid
	signature := helpers.Md5(subscriber.Email + uuid + subscriber.OutTime.Format("20060102150405")) //MD5加密
	subscriber.Signature = signature //密钥
	err := sendMail(subscriber.Email, "[YqStudio]邮箱验证", fmt.Sprintf("%s/active?sid=%s", system.GetConfiguration().Domain, signature))
	//如果邮箱验证成功
	if err == nil {
		//更新订阅者的信息
		err = subscriber.Update()
	}
	return err
}

//激活订阅
func ActiveSubsciber(c *gin.Context) {
	sid := c.Query("sid")
	var err error
	if len(sid) > 0 {
		var subscriber *models.Subscriber
		subscriber, err = models.GetSubscriberBySignature(sid) //根据订阅sid获取订阅者信息
		if err == nil {
			//如果订阅没有过期
			if helpers.GetCurrentTime().Before(subscriber.OutTime) {
				subscriber.VerifyState = true //设置验证状态为true
				subscriber.OutTime = helpers.GetCurrentTime() //超时时间为当前时间
				err = subscriber.Update() //更新订阅信息
				if err == nil {
					//向页面发送消息
					HandleMessage(c, "激活成功！")
					return
				}
			} else {
				err = errors.New("激活链接已过期，请重新获取！")
			}
		} else {
			err = errors.New("激活链接有误，请重新获取！")
		}
	} else {
		err = errors.New("激活链接有误，请重新获取！")
	}
	//向页面发送消息
	HandleMessage(c, err.Error())
}

//取消订阅
func UnSubscribe(c *gin.Context) {
	sid := c.Query("sid")
	if len(sid) > 0 {
		subscriber, err := models.GetSubscriberBySignature(sid)
		if err == nil && subscriber.VerifyState && subscriber.SubscribeState {
			subscriber.SubscribeState = false
			err = subscriber.Update()
			if err == nil {
				HandleMessage(c, "取消订阅成功!")
				return
			}
		}
	}
	HandleMessage(c, "Internal Server Error!")
}

func GetUnSubcribeUrl(subscriber *models.Subscriber) (string, error) {
	uuid := helpers.UUID()
	signature := helpers.Md5(subscriber.Email + uuid)
	subscriber.SecretKey = uuid
	subscriber.Signature = signature
	err := subscriber.Update()
	return fmt.Sprintf("%s/unsubscribe?sid=%s", system.GetConfiguration().Domain, signature), err
}

func sendEmailToSubscribers(subject, body string) error {
	subscribers, err := models.ListSubscriber(true)
	if err == nil {
		emails := make([]string, 0)
		for _, subscriber := range subscribers {
			emails = append(emails, subscriber.Email)
		}
		if len(emails) > 0 {
			err = sendMail(strings.Join(emails, ";"), subject, body)
		} else {
			err = errors.New("没有找到订阅者!")
		}
	}
	return err
}

func SubscriberIndex(c *gin.Context) {
	subscribers, _ := models.ListSubscriber(false)
	user, _ := c.Get(CONTEXT_USER_KEY)
	c.HTML(http.StatusOK, "admin/subscriber.html", gin.H{
		"subscribers": subscribers,
		"user":        user,
		"comments":    models.MustListUnreadComment(),
	})
}

// 邮箱为空时，发送给所有订阅者
func SubscriberPost(c *gin.Context) {
	mail := c.PostForm("mail")
	subject := c.PostForm("subject")
	body := c.PostForm("body")
	var err error
	if len(mail) > 0 {
		err = sendMail(mail, subject, body)
	} else {
		err = sendEmailToSubscribers(subject, body)
	}
	if err == nil {
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
