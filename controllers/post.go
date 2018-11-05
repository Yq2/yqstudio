package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"github.com/gin-gonic/gin"
	"github.com/Yq2/yqstudio/models"
)

func PostGet(c *gin.Context) {
	id := c.Param("id")
	post, err := models.GetPostById(id)
	if err == nil && post.IsPublished {
		post.View++
		post.UpdateView()
		post.Tags, _ = models.ListTagByPostId(id) //获取文章标签信息
		post.Comments, _ = models.ListCommentByPostID(id) //获取文章的评论
		user, _ := c.Get(CONTEXT_USER_KEY) //从session获取user
		c.HTML(http.StatusOK, "post/display.html", gin.H{
			"post": post, //文章信息
			"user": user, //用户
		})
	} else {
		//404
		Handle404(c)
	}
}

func PostNew(c *gin.Context) {
	c.HTML(http.StatusOK, "post/new.html", nil)
}

func PostCreate(c *gin.Context) {
	tags := c.PostForm("tags")
	title := c.PostForm("title")
	body := c.PostForm("body")
	isPublished := c.PostForm("isPublished")
	published := "on" == isPublished

	post := &models.Post{
		Title:       title,
		Body:        body,
		IsPublished: published,
	}
	err := post.Insert()
	if err == nil {
		if len(tags) > 0 {
			tagArr := strings.Split(tags, ",")
			for _, tag := range tagArr {
				tagId, err := strconv.ParseUint(tag, 10, 64)
				if err == nil {
					pt := &models.PostTag{
						PostId: post.ID,
						TagId:  uint(tagId),
					}
					pt.Insert()
				}
			}
		}
		c.Redirect(http.StatusMovedPermanently, "/admin/post")
	} else {
		c.HTML(http.StatusOK, "post/new.html", gin.H{
			"post":    post,
			"message": err.Error(),
		})
	}
}

func PostEdit(c *gin.Context) {
	id := c.Param("id")
	post, err := models.GetPostById(id)
	if err == nil {
		post.Tags, _ = models.ListTagByPostId(id)
		c.HTML(http.StatusOK, "post/modify.html", gin.H{
			"post": post,
		})
	} else {
		Handle404(c)
	}
}

func PostUpdate(c *gin.Context) {
	id := c.Param("id")
	tags := c.PostForm("tags")
	title := c.PostForm("title")
	body := c.PostForm("body")
	isPublished := c.PostForm("isPublished")
	published := "on" == isPublished

	pid, err := strconv.ParseUint(id, 10, 64)
	if err == nil {
		post := &models.Post{
			Title:       title,
			Body:        body,
			IsPublished: published,
		}
		post.ID = uint(pid)
		err = post.Update()
		if err == nil {
			// 删除tag
			models.DeletePostTagByPostId(post.ID)
			// 添加tag
			if len(tags) > 0 {
				tagArr := strings.Split(tags, ",")
				for _, tag := range tagArr {
					tagId, err := strconv.ParseUint(tag, 10, 64)
					if err == nil {
						pt := &models.PostTag{
							PostId: post.ID,
							TagId:  uint(tagId),
						}
						pt.Insert()
					}
				}
			}
			c.Redirect(http.StatusMovedPermanently, "/admin/post")
		} else {
			c.HTML(http.StatusOK, "post/modify.html", gin.H{
				"post":    post,
				"message": err.Error(),
			})
		}
	} else {
		Handle404(c)
	}
}

func PostPublish(c *gin.Context) {
	id := c.Param("id")
	post, err := models.GetPostById(id)
	if err == nil {
		post.IsPublished = !post.IsPublished
		err = post.Update()
	}
	c.JSON(http.StatusOK, gin.H{
		"succeed": err == nil,
	})
}

func PostDelete(c *gin.Context) {
	id := c.Param("id")
	pid, err := strconv.ParseUint(id, 10, 64)
	if err == nil {
		post := &models.Post{}
		post.ID = uint(pid)
		err = post.Delete()
		if err == nil {
			models.DeletePostTagByPostId(uint(pid))
			c.JSON(http.StatusOK, gin.H{
				"succeed": true,
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"succeed": false,
		"message": err.Error(),
	})
}

func PostIndex(c *gin.Context) {
	posts, _ := models.ListAllPost("") //根据tag获取文章列表
	user, _ := c.Get(CONTEXT_USER_KEY)
	c.HTML(http.StatusOK, "admin/post.html", gin.H{
		"posts":    posts, //所有文章
		"Active":   "posts",
		"user":     user,
		"comments": models.MustListUnreadComment(), //获取所有未读评论
	})
}
