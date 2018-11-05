package controllers

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"github.com/denisbakhtin/sitemap"
	"github.com/gin-gonic/gin"
	"github.com/Yq2/yqstudio/helpers"
	"github.com/Yq2/yqstudio/models"
	"github.com/Yq2/yqstudio/system"
)

const (
	SESSION_KEY          = "UserID"       // session key  用户ID
	CONTEXT_USER_KEY     = "User"         // context user key 用户信息
	SESSION_GITHUB_STATE = "GITHUB_STATE" // github state session key GitHub状态
	SESSION_CAPTCHA      = "GIN_CAPTCHA"  // captcha session key 验证码
)

func Handle404(c *gin.Context) {
	HandleMessage(c, "糟糕，页面找不到了")
}

func HandleMessage(c *gin.Context, message string) {
	c.HTML(http.StatusNotFound, "errors/error.html", gin.H{
		"message": message,
	})
}

func sendMail(to, subject, body string) error {
	c := system.GetConfiguration()
	return helpers.SendToMail(c.SmtpUsername, c.SmtpPassword, c.SmtpHost, to, subject, body, "html")
}

//查找配置中注册的邮箱账号 subject是标题， body表示内容
func NotifyEmail(subject, body string) error {
	//获取邮件通知地址
	notifyEmailsStr := system.GetConfiguration().NotifyEmails
	if notifyEmailsStr != "" {
		notifyEmails := strings.Split(notifyEmailsStr, ";") //可以一次性注册多个邮箱通知账号
		emails := make([]string, 0)
		for _, email := range notifyEmails {
			if email != "" {
				emails = append(emails, email)
			}
		}
		if len(emails) > 0 {
			//对每个注册邮箱进行通知
			return sendMail(strings.Join(emails, ";"), subject, body)
		}
	}
	return nil
}

/*func __sendMail(to, subject, body string) error {
	return nil
}*/
//网站地图
func CreateXMLSitemap() {
	configuration := system.GetConfiguration()
	folder := path.Join(configuration.Public, "sitemap")
	os.MkdirAll(folder, os.ModePerm)
	domain := configuration.Domain
	now := helpers.GetCurrentTime()
	items := make([]sitemap.Item, 0)

	items = append(items, sitemap.Item{
		Loc:        domain,
		LastMod:    now,
		Changefreq: "daily",
		Priority:   1,
	})
	//列出所有已经发布的文章
	posts, err := models.ListPublishedPost("")
	if err == nil {
		for _, post := range posts {
			items = append(items, sitemap.Item{
				Loc:        fmt.Sprintf("%s/post/%d", domain, post.ID),
				LastMod:    post.UpdatedAt,
				Changefreq: "weekly",
				Priority:   0.9,
			})
		}
	}
	//列出所有已经发布的页面
	pages, err := models.ListPublishedPage()
	if err == nil {
		for _, page := range pages {
			items = append(items, sitemap.Item{
				Loc:        fmt.Sprintf("%s/page/%d", domain, page.ID),
				LastMod:    page.UpdatedAt,
				Changefreq: "monthly",
				Priority:   0.8,
			})
		}
	}

	if err := sitemap.SiteMap(path.Join(folder, "sitemap1.xml.gz"), items); err != nil {
		return
	}
	if err := sitemap.SiteMapIndex(folder, "sitemap_index.xml", domain+"/static/sitemap/"); err != nil {
		return
	}
}
