package controllers

import (
	"fmt"
	"github.com/cihub/seelog" //日志库
	"github.com/gin-gonic/gin"
	"github.com/gorilla/feeds"  //rss订阅
	"github.com/Yq2/yqstudio/helpers"
	"github.com/Yq2/yqstudio/models"
	"github.com/Yq2/yqstudio/system"
)

func RssGet(c *gin.Context) {
	now := helpers.GetCurrentTime() //获取当前时间
	domain := system.GetConfiguration().Domain //获取主机地址
	feed := &feeds.Feed {
		Title:       "YqStudio",
		Link:        &feeds.Link{Href: domain},
		Description: "YqStudio, 分享关于你的一切",
		Author:      &feeds.Author{Name: "Yq", Email: "1225807604@qq.com"},
		Created:     now,
	}
	//构造一个feeds.Item 列表
	feed.Items = make([]*feeds.Item, 0)
	//查询 posts表和post_tags表
	posts, err := models.ListPublishedPost("") //查询所有发布文章以及标签
	if err == nil {
		for _, post := range posts {
			item := &feeds.Item {
				Id:          fmt.Sprintf("%s/post/%d", domain, post.ID),
				Title:       post.Title,
				Link:        &feeds.Link{Href: fmt.Sprintf("%s/post/%d", domain, post.ID)},
				Description: string(post.Excerpt()),
				Created:     now,
			}
			feed.Items = append(feed.Items, item)
		}
	}
	//生成订阅xml格式字符串
	rss, err := feed.ToRss()
	if err == nil {
		c.Writer.WriteString(rss)
	} else {
		//记录错误
		seelog.Error(err)
	}
}