package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday" //过滤HTML
	"github.com/russross/blackfriday"  //MarkdownCommon
	"github.com/Yq2/yqstudio/models"
	"net/http"
	"strconv"
)

func IndexGet(c *gin.Context) {
	//查询所有已经发布的文章
	posts, err := models.ListPublishedPost("")
	if err == nil {
		//构建HTML过滤政策
		policy := bluemonday.StrictPolicy()
		for _, post := range posts {
			//根据文章id获取文章所属标签tag
			post.Tags, _ = models.ListTagByPostId(strconv.FormatUint(uint64(post.ID), 10))
			//将文章body净化成受信任的HTML,并且是MarkdownCommon格式
			post.Body = policy.Sanitize(string(blackfriday.MarkdownCommon([]byte(post.Body))))
		}
		//获取user
		user, _ := c.Get(CONTEXT_USER_KEY)
		//渲染HTML，c.HTML
		c.HTML(http.StatusOK, "index/index.html", gin.H {
			"posts":    posts, //所有文章
			"tags":     models.MustListTag(), //所有标签(已经发布的文章的标签)
			"archives": models.MustListPostArchives(), //文章存档(查询某年某月一共发布了多少文章)
			"links":    models.MustListLinks(), //文章链接
			"user":     user, //来自session里面的user
		})
	} else {
		//中断请求，报500错误
		c.AbortWithStatus(http.StatusInternalServerError)
		//c.Abort()直接中断请求，不携带状态码
	}
}

//管理员首页
func AdminIndex(c *gin.Context) {
	//从session里面获取user
	user, _ := c.Get(CONTEXT_USER_KEY)
	c.HTML(http.StatusOK, "admin/index.html", gin.H {
		"pageCount":    models.CountPage(), //统计pages表
		"postCount":    models.CountPost(), //统计posts表
		"tagCount":     models.CountTag(), //统计tags表
		"commentCount": models.CountComment(), //统计comments表
		"user":         user, //来自session里面的user
		"comments":     models.MustListUnreadComment(), //查询所有未读评论
	})
}
