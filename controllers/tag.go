package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
	"github.com/Yq2/yqstudio/models"
	"net/http"
	"strconv"
)

func TagCreate(c *gin.Context) {
	name := c.PostForm("value")
	tag := &models.Tag{Name: name}
	err := tag.Insert()
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			"data": tag,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message": err.Error(),
		})
	}
}

func TagGet(c *gin.Context) {
	tagName := c.Param("tag")
	posts, err := models.ListPublishedPost(tagName) //根据标签名查找所有文章
	if err == nil {
		//构建一个过滤策略
		policy := bluemonday.StrictPolicy()
		for _, post := range posts {
			//根据文章id获取对应标签列表
			post.Tags, _ = models.ListTagByPostId(strconv.FormatUint(uint64(post.ID), 10))
			//将文章内容转为markdown 并进行安全过滤
			post.Body = policy.Sanitize(string(blackfriday.MarkdownCommon([]byte(post.Body))))
		}
		c.HTML(http.StatusOK, "index/index.html", gin.H{
			"posts":    posts,
			"tags":     models.MustListTag(),
			"archives": models.MustListPostArchives(), //文章的存档统计
			"links":    models.MustListLinks(), //所有链接
		})
	} else {
		//500服务器错误，并中断请求
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}
