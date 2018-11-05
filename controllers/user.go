package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/alimoeeny/gooauth2" //github回调处理
	"github.com/cihub/seelog"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/Yq2/yqstudio/helpers"
	"github.com/Yq2/yqstudio/models"
	"github.com/Yq2/yqstudio/system"
	"io/ioutil"
	"net/http"
	"strconv"
)

type GithubUserInfo struct {
	AvatarURL         string      `json:"avatar_url"`
	Bio               interface{} `json:"bio"`
	Blog              string      `json:"blog"`
	Company           interface{} `json:"company"`
	CreatedAt         string      `json:"created_at"`
	Email             interface{} `json:"email"`
	EventsURL         string      `json:"events_url"`
	Followers         int         `json:"followers"`
	FollowersURL      string      `json:"followers_url"`
	Following         int         `json:"following"`
	FollowingURL      string      `json:"following_url"`
	GistsURL          string      `json:"gists_url"`
	GravatarID        string      `json:"gravatar_id"`
	Hireable          interface{} `json:"hireable"`
	HTMLURL           string      `json:"html_url"`
	ID                int         `json:"id"`
	Location          interface{} `json:"location"`
	Login             string      `json:"login"`
	Name              interface{} `json:"name"`
	OrganizationsURL  string      `json:"organizations_url"`
	PublicGists       int         `json:"public_gists"`
	PublicRepos       int         `json:"public_repos"`
	ReceivedEventsURL string      `json:"received_events_url"`
	ReposURL          string      `json:"repos_url"`
	SiteAdmin         bool        `json:"site_admin"`
	StarredURL        string      `json:"starred_url"`
	SubscriptionsURL  string      `json:"subscriptions_url"`
	Type              string      `json:"type"`
	UpdatedAt         string      `json:"updated_at"`
	URL               string      `json:"url"`
}

func SigninGet(c *gin.Context) {
	c.HTML(http.StatusOK, "auth/signin.html", nil)
}

func SignupGet(c *gin.Context) {
	c.HTML(http.StatusOK, "auth/signup.html", nil)
}

func LogoutGet(c *gin.Context) {
	s := sessions.Default(c)
	//清空session
	s.Clear()
	//保存session
	s.Save()
	//重定向到登录页面
	c.Redirect(http.StatusSeeOther, "/signin")
}

//注册
func SignupPost(c *gin.Context) {
	email := c.PostForm("email")
	telephone := c.PostForm("telephone")
	password := c.PostForm("password")
	seelog.Debugf("email:%s,telephone:%s,password:%s",email,telephone,password)
	user := &models.User{
		Email:     email,
		Telephone: telephone,
		Password:  password,
		IsAdmin:   true,
	}
	var err error
	if len(user.Email) == 0 || len(user.Password) == 0 {
		err = errors.New("邮箱或者密码不能为空.")
	} else {
		//MD5加密
		user.Password = helpers.Md5(user.Email + user.Password)
		user.OutTime = helpers.GetCurrentTime() //过期时间不能为空
		seelog.Debugf("md5 Password:%s",user.Password)
		err = user.Insert()
		if err == nil {
			c.JSON(http.StatusOK, gin.H{
				"succeed": true,
			})
			return
		} else {
			seelog.Debugf("email is exist err:%s",err.Error())
			err = errors.New("注册邮箱已存在.")
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"succeed": false,
		"message": err.Error(),
	})
}
//登录
func SigninPost(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	var err error
	if len(username) > 0 && len(password) > 0 {
		var user *models.User
		user, err = models.GetUserByUsername(username)
		fmt.Println(user, err)
		if err == nil && user.Password == helpers.Md5(username+password) {
			//如果用户状态没有被锁定
			if !user.LockState {
				//从gin里面取出session
				s := sessions.Default(c)
				//清空当前客户端的session
				s.Clear()
				//设置UserId
				s.Set(SESSION_KEY, user.ID)
				//保存session
				s.Save()
				if user.IsAdmin {
					//301跳转到管理员首页
					c.Redirect(http.StatusMovedPermanently, "/admin/index")
				} else {
					//301跳转到普通首页
					c.Redirect(http.StatusMovedPermanently, "/")
				}
				return
			} else {
				err = errors.New("糟糕，你的账户被锁定了.")
			}
		} else {
			err = errors.New("用户名无效或者密码不正确.")
		}
	} else {
		err = errors.New("用户名或者密码不能为空.")
	}
	c.HTML(http.StatusOK, "auth/signin.html", gin.H{
		"message": err.Error(),
	})
}

func Oauth2Callback(c *gin.Context) {
	code := c.Query("code") //url后面的?code=xxx&state=yyy
	state := c.Query("state")
	//取出session
	session := sessions.Default(c)
	//如果session里面没有 GITHUB_STATE 字段
	if len(state) == 0 || state != session.Get(SESSION_GITHUB_STATE) {
		//中止请求
		c.Abort()
		return
	} else {
		//删除 GITHUB_STATE
		session.Delete(SESSION_GITHUB_STATE)
		session.Save()
	}
	//将回调的code转换成token
	token, err := exchangeTokenByCode(code)
	if err == nil {
		var userInfo *GithubUserInfo
		//根据token 翻译成userinfo
		userInfo, err = getGithubUserInfoByAccessToken(token)
		if err == nil {
			var user *models.User
			//从session里面获取user
			if sessionUser, exists := c.Get(CONTEXT_USER_KEY); exists {
				//断言session里面的用户是否是model.User类型
				user, _ = sessionUser.(*models.User)
				//检测 userid是否是token解析出来的用户
				_, err1 := models.IsGithubIdExists(userInfo.Login, user.ID)
				if err1 != nil { // 未绑定
					if user.IsAdmin {
						user.GithubLoginId = userInfo.Login
					}
					user.AvatarUrl = userInfo.AvatarURL
					user.GithubUrl = userInfo.HTMLURL
					//更新保存用户信息
					err = user.UpdateGithubUserInfo()
				} else {
					//token翻译过来的用户信息和当前session里面的userid不匹配
					err = errors.New("当前GitHub ID不能绑定另一个账户.")
				}
				//用户连接的session里面没有userid
			} else {
				//根据翻译过来的UserInfo，构造user
				user = &models.User{
					GithubLoginId: userInfo.Login,
					AvatarUrl:     userInfo.AvatarURL,
					GithubUrl:     userInfo.HTMLURL,
				}
				user, err = user.FirstOrCreate()
				if err == nil {
					//用户状态被锁定，这种情况说明数据库已经存在这个token解析过来的用户信息
					if user.LockState {
						err = errors.New("糟糕，你的账户被锁定了.")
						HandleMessage(c, "糟糕，你的账户被锁定了.")
						return
					}
				}
			}
			//检测上面每一步是否出错
			if err == nil {
				s := sessions.Default(c)
				s.Clear()
				s.Set(SESSION_KEY, user.ID)
				//更新保存用户session
				s.Save()
				//是管理员
				if user.IsAdmin {
					c.Redirect(http.StatusMovedPermanently, "/admin/index")
				} else {
					c.Redirect(http.StatusMovedPermanently, "/")
				}
				return
			}
		}
	}
	//记录错误日志
	seelog.Error(err)
	//GitHub token解析不通过，301跳转到重新登录
	c.Redirect(http.StatusMovedPermanently, "/signin")
}

func exchangeTokenByCode(code string) (string, error) {
	t := &oauth.Transport{Config: &oauth.Config{
		ClientId:     system.GetConfiguration().GithubClientId, //id
		ClientSecret: system.GetConfiguration().GithubClientSecret, //密钥
		RedirectURL:  system.GetConfiguration().GithubRedirectURL, //回调地址，这里是当前服务callback地址
		TokenURL:     system.GetConfiguration().GithubTokenUrl, //token验证URL
		Scope:        system.GetConfiguration().GithubScope, //GitHub 命名空间
	}}
	//将code转换成token
	tok, err := t.Exchange(code)
	if err == nil {
		//创建一个token缓存文件
		tokenCache := oauth.CacheFile("./request.token")
		//将生成的token存入缓存文件
		err := tokenCache.PutToken(tok)
		//返回接入token
		return tok.AccessToken, err
	}
	return "", err
}
//将token解析成GitHub用户信息
func getGithubUserInfoByAccessToken(token string) (*GithubUserInfo, error) {
	//请求GitHub token验证url
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/user?access_token=%s", token))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var body []byte
	//将body的内容全部读出来
	body, err = ioutil.ReadAll(resp.Body)
	if err == nil {
		var userInfo GithubUserInfo
		//将body反序列化成对象
		err = json.Unmarshal(body, &userInfo)
		return &userInfo, err
	}
	return nil, err
}

//GET 用户数据文件 profile.html
func ProfileGet(c *gin.Context) {
	//从session里面获取user
	sessionUser, exists := c.Get(CONTEXT_USER_KEY)
	//如果存在 将用户数据渲染到HTML
	if exists {
		c.HTML(http.StatusOK, "admin/profile.html", gin.H {
			"user":     sessionUser,
			//用户的评论
			"comments": models.MustListUnreadComment(),
		})
	}
}

//更新用户数据保存文件
func ProfileUpdate(c *gin.Context) {
	avatarUrl := c.PostForm("avatarUrl")
	nickName := c.PostForm("nickName")
	sessionUser, _ := c.Get(CONTEXT_USER_KEY)
	if user, ok := sessionUser.(*models.User); ok {
		err := user.UpdateProfile(avatarUrl, nickName)
		if err == nil {
			//响应json
			c.JSON(http.StatusOK, gin.H {
				"succeed": true,
				"user":    models.User{AvatarUrl: avatarUrl, NickName: nickName},
			})
		} else {
			c.JSON(http.StatusOK, gin.H {
				"succeed": false,
				"message": err.Error(),
			})
		}
	}
}
//绑定邮箱
func BindEmail(c *gin.Context) {
	//从post参数获取email字段
	email := c.PostForm("email")
	sessionUser, _ := c.Get(CONTEXT_USER_KEY)
	if user, ok := sessionUser.(*models.User); ok {
		//如果session里面的email字段不为空，说明session里面的user已经绑定了email
		if len(user.Email) > 0 {
			c.JSON(http.StatusOK, gin.H {
				"succeed": false,
				"message": "不能重复绑定邮箱.",
			})
		} else {
			//根据email获取用户信息，users表里面有email这个字段
			_, err := models.GetUserByUsername(email)
			if err != nil {
				//把user里面的email字段更新
				err := user.UpdateEmail(email)
				c.JSON(http.StatusOK, gin.H {
					"succeed": err == nil,
				})
			} else {
				c.JSON(http.StatusOK, gin.H {
					"succeed": false,
					"message": "邮箱没有被验证!",
				})
			}
		}
	}
}

//解除绑定email
func UnbindEmail(c *gin.Context) {
	//从用户session里面获取user
	sessionUser, _ := c.Get(CONTEXT_USER_KEY)
	//类型断言sessionUser是否是model.User的类型，如果断言成功那么sessionUser就是*model.User的动态类型
	if user, ok := sessionUser.(*models.User); ok {
		if len(user.Email) == 0 {
			c.JSON(http.StatusOK, gin.H {
				"succeed": false,
				"message": "邮箱不能被解绑.",
			})
		} else {
			err := user.UpdateEmail("")
			c.JSON(http.StatusOK, gin.H {
				"succeed": err == nil,
			})
		}
	}
}
//解绑GitHub账号
func UnbindGithub(c *gin.Context) {
	//从session里面获取User
	sessionUser, _ := c.Get(CONTEXT_USER_KEY)
	if user, ok := sessionUser.(*models.User); ok {
		//如果user里面GitHub为空，则说明该用户并没有绑定GitHub
		if len(user.GithubLoginId) == 0 {
			c.JSON(http.StatusOK, gin.H {
				"succeed": false,
				"message": "GitHub 账户不能被解绑.",
			})
		} else {
			//将user里面的GitHubid置空
			user.GithubLoginId = ""
			//更新用户信息
			err := user.UpdateGithubUserInfo()
			c.JSON(http.StatusOK, gin.H {
				"succeed": err == nil,
			})
		}
	}
}
//获取用户列表
func UserIndex(c *gin.Context) {
	//从数据库里面加载所有用户
	users, _ := models.ListUsers()
	//从session里面获取当前登录用户
	user, _ := c.Get(CONTEXT_USER_KEY)
	c.HTML(http.StatusOK, "admin/user.html", gin.H {
		"users":    users,
		"user":     user,
		"comments": models.MustListUnreadComment(), //获取所有未读评论
	})
}
//执行用户锁定
func UserLock(c *gin.Context) {
	id := c.Param("id") //quey == ?id=123
	_id, _ := strconv.ParseUint(id, 10, 64)
	//根据用户ID查询用户信息
	user, err := models.GetUser(uint(_id))
	if err == nil {
		//改变用户锁定状态为锁定状态
		user.LockState = !user.LockState
		//将锁定状态更新到数据库
		err = user.Lock()
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
