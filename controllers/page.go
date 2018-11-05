package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/Yq2/yqstudio/models"
	"net/http"
	"strconv"
)

func PageGet(c *gin.Context) {
	id := c.Param("id")
	//根据ID获取页面page
	page, err := models.GetPageById(id)
	//如果页面是已经发布的
	if err == nil && page.IsPublished {
		page.View++ //view字段自增
		page.UpdateView() //更新page的view字段
		c.HTML(http.StatusOK, "page/display.html", gin.H{
			"page": page,
		})
	} else {
		//404
		Handle404(c)
	}
}
//渲染页面
func PageNew(c *gin.Context) {
	c.HTML(http.StatusOK, "page/new.html", nil)
}

func PageCreate(c *gin.Context) {
	title := c.PostForm("title")
	body := c.PostForm("body")
	isPublished := c.PostForm("isPublished")
	published := "on" == isPublished
	page := &models.Page{
		Title:       title,
		Body:        body,
		IsPublished: published,
	}
	//保存一篇文章
	err := page.Insert()
	if err == nil {
		//301重定向
		c.Redirect(http.StatusMovedPermanently, "/admin/page")
	} else {
		c.HTML(http.StatusOK, "page/new.html", gin.H{
			"message": err.Error(),
			"page":    page,
		})
	}
}

func PageEdit(c *gin.Context) {
	id := c.Param("id")
	page, err := models.GetPageById(id)
	if err == nil {
		c.HTML(http.StatusOK, "page/modify.html", gin.H{
			"page": page,
		})
	} else {
		Handle404(c)
	}
}

func PageUpdate(c *gin.Context) {
	id := c.Param("id")
	title := c.PostForm("title")
	body := c.PostForm("body")
	isPublished := c.PostForm("isPublished")
	published := "on" == isPublished
	pid, err := strconv.ParseUint(id, 10, 64)
	if err == nil {
		page := &models.Page{Title: title, Body: body, IsPublished: published}
		page.ID = uint(pid)
		err = page.Update()
		if err == nil {
			c.Redirect(http.StatusMovedPermanently, "/admin/page")
			return
		}
	}
	c.AbortWithError(http.StatusInternalServerError, err)
}

func PagePublish(c *gin.Context) {
	id := c.Param("id")
	page, err := models.GetPageById(id)
	if err == nil {
		page.IsPublished = !page.IsPublished
		err = page.Update()
	}
	c.JSON(http.StatusOK, gin.H{
		"succeed": err == nil,
	})
}

func PageDelete(c *gin.Context) {
	id := c.Param("id")
	pid, err := strconv.ParseUint(id, 10, 64)
	if err == nil {
		page := &models.Page{}
		page.ID = uint(pid)
		//删除page
		err = page.Delete()
		if err == nil {
			c.JSON(http.StatusOK, gin.H{
				"succeed": true,
			})
			return
		}
	}
	c.AbortWithError(http.StatusInternalServerError, err)
}

func PageIndex(c *gin.Context) {
	pages, _ := models.ListAllPage() //加载所有页面
	user, _ := c.Get(CONTEXT_USER_KEY) //从session获取user
	c.HTML(http.StatusOK, "admin/page.html", gin.H{
		"pages":    pages,
		"user":     user,
		"comments": models.MustListUnreadComment(), //获取所有未读评论列表
	})
}
