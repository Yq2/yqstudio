package main

import (
	"flag"
	"html/template"
	"net/http"
	"path/filepath"
	"github.com/cihub/seelog" //日志库
	"github.com/claudiu/gocron"
	"github.com/gin-contrib/sessions"  //gin session管理:有基于cookie，基于Redis，基于内存，基于Mongodb
	//"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	//_ "github.com/go-redis/redis"
	"github.com/Yq2/yqstudio/controllers"
	"github.com/Yq2/yqstudio/helpers"
	"github.com/Yq2/yqstudio/models"
	"github.com/Yq2/yqstudio/system"
)

func main() {

	configFilePath := flag.String("C", "conf/conf.yaml", "config file path")
	logConfigPath := flag.String("L", "conf/seelog.xml", "log config file path")
	flag.Parse()
	//加载日志配置文件，并生成logger对象
	logger, err := seelog.LoggerFromConfigAsFile(*logConfigPath)
	if err != nil {
		//严重错误
		seelog.Critical("err parsing seelog config file", err)
		return
	}
	//替换日志日志配配置
	seelog.ReplaceLogger(logger)
	//main函数运行结束时将日志缓冲区的数据写入文件
	defer seelog.Flush()
	//加载配置文件
	if err := system.LoadConfiguration(*configFilePath); err != nil {
		seelog.Critical("err parsing config log file", err)
		return
	}
	//初始化数据库连接
	db, err := models.InitDB()
	if err != nil {
		//严重错误
		seelog.Critical("err open databases", err)
		return
	}
	defer db.Close()
	//release 设置gin模式为线上模式
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	//配置模板解析函数，设置模板引擎
	setTemplate(router)
	//配置session管理
	setSessions(router)
	//配置用户登录 注册
	router.Use(SharedData())

	//Periodic tasks
	//定时任务触发
	//Day 将工作单位设置为天数
	//Days 以周为单位，间隔为1
	gocron.Every(1).Day().Do(controllers.CreateXMLSitemap) //创建xml网站地图
	gocron.Every(7).Days().Do(controllers.Backup) //定时备份
	//开启定时任务
	gocron.Start()
	//挂载静态目录
	router.Static("/static", filepath.Join(getCurrentDirectory(), "./static"))
	//定义找不到路由处理情况
	router.NoRoute(controllers.Handle404)
	router.GET("/", controllers.IndexGet)
	router.GET("/index", controllers.IndexGet)
	router.GET("/rss", controllers.RssGet)

	if system.GetConfiguration().SignupEnabled {
		//渲染退出登录页面
		router.GET("/signup", controllers.SignupGet)
		//提交退出登录POST请求
		router.POST("/signup", controllers.SignupPost)
	}
	// user signin and logout
	router.GET("/signin", controllers.SigninGet)
	router.POST("/signin", controllers.SigninPost)
	router.GET("/logout", controllers.LogoutGet) //清空session
	router.GET("/oauth2callback", controllers.Oauth2Callback) //GitHub认证回调处理
	router.GET("/auth/:authType", controllers.AuthGet)

	// captcha 生成图片验证码，保存在session，并将图片编码写入响应体
	router.GET("/captcha", controllers.CaptchaGet)

	visitor := router.Group("/visitor") //游客路由
	//用户认证中间件
	visitor.Use(AuthRequired())
	{
		visitor.POST("/new_comment", controllers.CommentPost) //发布文章评论，并邮件通知预置邮箱
		visitor.POST("/comment/:id/delete", controllers.CommentDelete)
	}

	// subscriber
	//  订阅
	router.GET("/subscribe", controllers.SubscribeGet) //获取订阅统计
	router.POST("/subscribe", controllers.Subscribe) //订阅
	router.GET("/active", controllers.ActiveSubsciber) //激活订阅
	router.GET("/unsubscribe", controllers.UnSubscribe)  //取消订阅

	router.GET("/page/:id", controllers.PageGet) //页面访问，view自增
	router.GET("/post/:id", controllers.PostGet) //请求文章以及评论
	router.GET("/tag/:tag", controllers.TagGet) //根据标签名获取文章
	router.GET("/archives/:year/:month", controllers.ArchiveGet) //获取指定年-月的存档文章

	router.GET("/link/:id", controllers.LinkGet) //获取链接，并且view字段自增

	authorized := router.Group("/admin") //admin组
	//管理员认证中间件
	authorized.Use(AdminScopeRequired())
	{
		// index
		authorized.GET("/index", controllers.AdminIndex)

		// image upload
		authorized.POST("/upload", controllers.Upload)

		// page
		authorized.GET("/page", controllers.PageIndex) //
		authorized.GET("/new_page", controllers.PageNew) //渲染新建页面
		authorized.POST("/new_page", controllers.PageCreate) //提交一篇page
		authorized.GET("/page/:id/edit", controllers.PageEdit) //渲染编辑页面
		authorized.POST("/page/:id/edit", controllers.PageUpdate) //编辑页面
		authorized.POST("/page/:id/publish", controllers.PagePublish) //发布页面
		authorized.POST("/page/:id/delete", controllers.PageDelete) //删除page

		// post
		authorized.GET("/post", controllers.PostIndex)
		authorized.GET("/new_post", controllers.PostNew)
		authorized.POST("/new_post", controllers.PostCreate)
		authorized.GET("/post/:id/edit", controllers.PostEdit)
		authorized.POST("/post/:id/edit", controllers.PostUpdate)
		authorized.POST("/post/:id/publish", controllers.PostPublish)
		authorized.POST("/post/:id/delete", controllers.PostDelete)

		// tag
		authorized.POST("/new_tag", controllers.TagCreate)

		//
		authorized.GET("/user", controllers.UserIndex)
		authorized.POST("/user/:id/lock", controllers.UserLock) //锁定用户

		// profile
		//配置文件
		authorized.GET("/profile", controllers.ProfileGet)
		authorized.POST("/profile", controllers.ProfileUpdate)
		authorized.POST("/profile/email/bind", controllers.BindEmail) //绑定email
		authorized.POST("/profile/email/unbind", controllers.UnbindEmail) //解绑email
		authorized.POST("/profile/github/unbind", controllers.UnbindGithub) //解绑GitHub

		// subscriber
		//订阅者管理
		authorized.GET("/subscriber", controllers.SubscriberIndex)
		authorized.POST("/subscriber", controllers.SubscriberPost) //将文章推送给订阅者

		// link
		authorized.GET("/link", controllers.LinkIndex) //获取链接页面
		authorized.POST("/new_link", controllers.LinkCreate) //创建链接
		authorized.POST("/link/:id/edit", controllers.LinkUpdate) //更新链接
		authorized.POST("/link/:id/delete", controllers.LinkDelete) //删除链接

		// comment
		authorized.POST("/comment/:id", controllers.CommentRead) //更新评论
		authorized.POST("/read_all", controllers.CommentReadAll) //将所有未读评论设置为已读评论

		// backup
		authorized.POST("/backup", controllers.BackupPost) //加密备份，将文件上传到七牛
		authorized.POST("/restore", controllers.RestorePost) //从七牛下载文件到本地

		// mail
		authorized.POST("/new_mail", controllers.SendMail) //向指定ID订阅者发送邮件
		authorized.POST("/new_batchmail", controllers.SendBatchMail) //批量向订阅者发送邮件
	}
	//启动web服务
	router.Run(system.GetConfiguration().Addr)
	//开启HTTPS
	//var serverCrt string = filepath.Join(getCurrentDirectory(), "./static/tls/server.crt")
	//var serverKey string = filepath.Join(getCurrentDirectory(), "./static/tls/server.key")
	//router.RunTLS(system.GetConfiguration().Addr, serverCrt, serverKey)
}

func setTemplate(engine *gin.Engine) {

	funcMap := template.FuncMap {
		"dateFormat": helpers.DateFormat,
		"substring":  helpers.Substring,
		"isOdd":      helpers.IsOdd,
		"isEven":     helpers.IsEven,
		"truncate":   helpers.Truncate,
		"add":        helpers.Add,
		"listtag":    helpers.ListTag,
	}
	//为gin 引擎设置解析函数
	engine.SetFuncMap(funcMap)
	//加载模板引擎 == 当前运行目录 + "./views/**/*"
	engine.LoadHTMLGlob(filepath.Join(getCurrentDirectory(), "./views/**/*"))
}

//setSessions initializes sessions & csrf middlewares
func setSessions(router *gin.Engine) {
	config := system.GetConfiguration()
	//https://github.com/gin-gonic/contrib/tree/master/sessions
	store := sessions.NewCookieStore([]byte(config.SessionSecret))
	//store, _ := redis.NewStore(9, "tcp", config.RedisHost, "", []byte(config.SessionSecret))
	store.Options(sessions.Options{HttpOnly: true, MaxAge: 7 * 86400, Path: "/"}) //Also set Secure: true if using SSL, you should though
	//当前引擎使用session 名字为gin-session
	router.Use(sessions.Sessions("gin-session", store))
	//https://github.com/utrack/gin-csrf
	/*router.Use(csrf.Middleware(csrf.Options{
		Secret: config.SessionSecret,
		ErrorFunc: func(c *gin.Context) {
			c.String(400, "CSRF token mismatch")
			c.Abort()
		},
	}))*/
}

//+++++++++++++ middlewares +++++++++++++++++++++++

//SharedData fills in common data, such as user info, etc...
//根据session里面的 UserID 加载对应用户信息
func SharedData() gin.HandlerFunc {
	return func(c *gin.Context) {
		//从gin引擎中获取默认的session
		session := sessions.Default(c)
		//UserID
		if uID := session.Get(controllers.SESSION_KEY); uID != nil {
			//获取用户数据
			user, err := models.GetUser(uID)
			if err == nil {
				//如果存在该用户,将User字段设置为从数据库查询的user
				c.Set(controllers.CONTEXT_USER_KEY, user)
			}
		}
		//配置是否已注册
		if system.GetConfiguration().SignupEnabled {
			c.Set("SignupEnabled", true)
		}
		//将中间件流转
		c.Next()
	}
}

//AuthRequired grants access to authenticated users, requires SharedData middleware
//管理员认证中间件
func AdminScopeRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		//获取User
		if user, _ := c.Get(controllers.CONTEXT_USER_KEY); user != nil {
			//管理员验证
			if u, ok := user.(*models.User); ok && u.IsAdmin {
				c.Next()
				return
			}
		}
		seelog.Warnf("User not authorized to visit %s", c.Request.RequestURI)
		//403
		c.HTML(http.StatusForbidden, "errors/error.html", gin.H{
			"message": "会话失效!",
		})
		c.Abort()
	}
}

//验证用户是否已登录
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if user, _ := c.Get(controllers.CONTEXT_USER_KEY); user != nil {
			//验证用户是否登录
			if _, ok := user.(*models.User); ok {
				c.Next()
				return
			}
		}
		seelog.Warnf("User not authorized to visit %s", c.Request.RequestURI)
		//响应403请求
		c.HTML(http.StatusForbidden, "errors/error.html", gin.H{
			"message": "会话失效!",
		})
		//中断后续请求
		c.Abort()
	}
}

//func getCurrentDirectory() string {
//	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
//	if err != nil {
//		seelog.Critical(err)
//	}
//	return strings.Replace(dir, "\\", "/", -1)
//}

func getCurrentDirectory() string {
	return ""
}
