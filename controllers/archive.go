package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
	"github.com/Yq2/yqstudio/models"
	"net/http"
	"strconv"
)

func ArchiveGet(c *gin.Context) {
	year := c.Param("year")
	month := c.Param("month")
	//根据年-月信息获取存档文章(统计在指定年，指定月的文章列表)
	posts, err := models.ListPostByArchive(year, month)
	if err == nil {
		policy := bluemonday.StrictPolicy()
		for _, post := range posts {
			post.Tags, _ = models.ListTagByPostId(strconv.FormatUint(uint64(post.ID), 10))
			//净化html
			post.Body = policy.Sanitize(string(blackfriday.MarkdownCommon([]byte(post.Body))))
		}
		c.HTML(http.StatusOK, "index/index.html", gin.H{
			"posts":    posts,
			"tags":     models.MustListTag(),
			"archives": models.MustListPostArchives(),
			"links":    models.MustListLinks(),
		})
	} else {
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}
