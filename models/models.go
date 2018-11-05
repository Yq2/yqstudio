package models

import (
	"database/sql"
	"fmt"
	"html/template"
	"strconv"
	"time"
	"github.com/jinzhu/gorm"
	//_ "github.com/mattn/go-sqlite3"
	_ "github.com/go-sql-driver/mysql"
	"github.com/microcosm-cc/bluemonday" //过滤不信任的内容
	"github.com/russross/blackfriday" //markdown解析库
	"github.com/Yq2/yqstudio/system"
)

// I don't need soft delete,so I use customized BaseModel instead gorm.Model
type BaseModel struct {
	ID        uint `gorm:"primary_key"`  //int(10) unsigned
	CreatedAt time.Time //timestamp
	UpdatedAt time.Time //timestamp
}

// table pages
type Page struct {
	BaseModel
	Title       string // title //varchar(255)
	Body        string // body //varchar(255)
	View        int    // view count //int(11)
	IsPublished bool   // published or not //tinyint(1)
}

// table posts
type Post struct {
	BaseModel
	Title       string     // title //varchar(255)
	Body        string     // body //varchar(255)
	View        int        // view count //int(11)
	IsPublished bool       // published or not  //tinyint(1)
	Tags        []*Tag     `gorm:"-"` // tags of post //忽略这个字段
	Comments    []*Comment `gorm:"-"` // comments of post //忽略这个字段
}

// table tags
type Tag struct {
	BaseModel
	Name  string // tag name  //varchar(255)
	Total int    `gorm:"-"` // count of post //忽略这个字段
}

// table post_tags
type PostTag struct {
	BaseModel
	PostId uint // post id //int(10) unsigned
	TagId  uint // tag id //int(10) unsigned
}

// table users
type User struct {
	gorm.Model
	Email         string    `gorm:"unique_index;default:null"` //邮箱 //varchar(255)
	Telephone     string    `gorm:"unique_index;default:null"` //手机号码 //varchar(255)
	Password      string    `gorm:"default:null"`              //密码  //varchar(255)
	VerifyState   string    `gorm:"default:'0'"`               //邮箱验证状态 //varchar(255)
	SecretKey     string    `gorm:"default:null"`              //密钥 //varchar(255)
	OutTime       time.Time //过期时间  timestamp
	GithubLoginId string    `gorm:"unique_index;default:null"` // github唯一标识 //varchar(255)
	GithubUrl     string    //github地址 varchar(255) //varchar(255)
	IsAdmin       bool      //是否是管理员 tinyint(1)
	AvatarUrl     string    // 头像链接  //varchar(255)
	NickName      string    // 昵称  //varchar(255)
	LockState     bool      `gorm:"default:'0'"` //锁定状态tinyint(1)
}

// table comments
type Comment struct {
	BaseModel
	UserID    uint   // 用户id timestamp
	Content   string // 内容 varchar(255)
	PostID    uint   // 文章id int(10) unsigned
	ReadState bool   `gorm:"default:'0'"` // 阅读状态 tinyint(1)
	//Replies []*Comment // 评论
	NickName  string `gorm:"-"` //忽略这个字段
	AvatarUrl string `gorm:"-"` //忽略这个字段
	GithubUrl string `gorm:"-"` //忽略这个字段
}

// table subscribe
type Subscriber struct {
	gorm.Model
	Email          string    `gorm:"unique_index"` //邮箱
	VerifyState    bool      `gorm:"default:'0'"`  //验证状态
	SubscribeState bool      `gorm:"default:'1'"`  //订阅状态
	OutTime        time.Time //过期时间
	SecretKey      string    // 秘钥
	Signature      string    //签名
}

// table link
type Link struct {
	gorm.Model
	Name string //名称
	Url  string //地址
	Sort int    `gorm:"default:'0'"` //排序
	View int    //访问次数
}

// query result
type QrArchive struct {
	ArchiveDate time.Time //month
	Total       int       //total
	Year        int       // year
	Month       int       // month
}

var DB *gorm.DB

func InitDB() (*gorm.DB, error) {

	db, err := gorm.Open("mysql", system.GetConfiguration().DSN)
	//db, err := gorm.Open("mysql", "root:mysql@/wblog?charset=utf8&parseTime=True&loc=Asia/Shanghai")
	if err == nil {
		DB = db
		db.LogMode(true) //开启日志打印
		db.AutoMigrate(&Page{}, &Post{}, &Tag{}, &PostTag{}, &User{}, &Comment{}, &Subscriber{}, &Link{}) //自动创建所有表
		db.Model(&PostTag{}).AddUniqueIndex("uk_post_tag", "post_id", "tag_id") //给表添加唯一索引（组合索引）
		return db, err
	}
	return nil, err
}

// Page
func (page *Page) Insert() error {
	return DB.Create(page).Error //.Error可以返回执行过程中的错误
}

func (page *Page) Update() error {
	return DB.Model(page).Updates(map[string]interface{}{
		"title":        page.Title,
		"body":         page.Body,
		"is_published": page.IsPublished,
	}).Error //返回执行过程中的错误
}

func (page *Page) UpdateView() error {
	return DB.Model(page).Updates(map[string]interface{}{
		"view": page.View,
	}).Error
}

func (page *Page) Delete() error {
	return DB.Delete(page).Error
}

func GetPageById(id string) (*Page, error) {
	//parseUint将字符串转换成int
	pid, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, err
	}
	var page Page
	//查询主键ID并返回执行过程的错误
	err = DB.First(&page, "id = ?", pid).Error
	return &page, err
}

func ListPublishedPage() ([]*Page, error) {
	return _listPage(true)
}

func ListAllPage() ([]*Page, error) {
	return _listPage(false)
}

func _listPage(published bool) ([]*Page, error) {
	var pages []*Page
	var err error
	if published {
		//where构造查询条件
		err = DB.Where("is_published = ?", true).Find(&pages).Error
	} else {
		//直接查询
		err = DB.Find(&pages).Error
	}
	return pages, err
}

func CountPage() int {
	var count int
	//查询 pages表 所有记录统计计数
	DB.Model(&Page{}).Count(&count)
	return count
}

// Post
func (post *Post) Insert() error {
	//插入一条记录
	return DB.Create(post).Error
}

func (post *Post) Update() error {
	//update是有变化才更新，save是不管是否有变化都保存
	return DB.Model(post).Updates(map[string]interface{}{
		"title":        post.Title,
		"body":         post.Body,
		"is_published": post.IsPublished,
	}).Error
}

func (post *Post) UpdateView() error {
	return DB.Model(post).Updates(map[string]interface{}{
		"view": post.View,
	}).Error
}

func (post *Post) Delete() error {
	return DB.Delete(post).Error
}

func (post *Post) Excerpt() template.HTML {
	//you can sanitize, cut it down, add images, etc
	policy := bluemonday.StrictPolicy() //remove all html tags
	sanitized := policy.Sanitize(string(blackfriday.MarkdownCommon([]byte(post.Body))))
	runes := []rune(sanitized)
	//将多余内容切掉
	if len(runes) > 300 {
		sanitized = string(runes[:300])
	}
	//将内容输出到网页
	excerpt := template.HTML(sanitized + "...")
	return excerpt
}

func ListPublishedPost(tag string) ([]*Post, error) {
	return _listPost(tag, true)
}

func ListAllPost(tag string) ([]*Post, error) {
	return _listPost(tag, false)
}

func _listPost(tag string, published bool) ([]*Post, error) {
	var posts []*Post
	var err error
	if len(tag) > 0 {
		tagId, err := strconv.ParseUint(tag, 10, 64)
		if err != nil {
			return nil, err
		}
		var rows *sql.Rows
		if published {
			//DB.Raw()执行原生SQL 查询文章表和文章标签表
			rows, err = DB.Raw("select p.* from posts p inner join post_tags pt on p.id = pt.post_id where pt.tag_id = ? and p.is_published = ? order by created_at desc", tagId, true).Rows()
		} else {
			//DB.Raw()执行原生SQL
			rows, err = DB.Raw("select p.* from posts p inner join post_tags pt on p.id = pt.post_id where pt.tag_id = ? order by created_at desc", tagId).Rows()
		}
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		//依次读取查询到的每一行
		for rows.Next() {
			var post Post
			//将读取到的行扫描到post结构中
			DB.ScanRows(rows, &post)
			//将每一行结构化的结果加入到slice
			posts = append(posts, &post)
		}
	} else {
		if published {
			//加上查询条件然后按照创建时间倒序排列，最后返回执行过程中的错误
			err = DB.Where("is_published = ?", true).Order("created_at desc").Find(&posts).Error
		} else {
			//只是按照创建时间降序排列，最后返回执行过程中的错误
			err = DB.Order("created_at desc").Find(&posts).Error
		}
	}
	return posts, err
}

func CountPost() int {
	var count int
	//统计post表所有记录
	DB.Model(&Post{}).Count(&count)
	return count
}

func GetPostById(id string) (*Post, error) {
	pid, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, err
	}
	var post Post
	//查询第一条记录
	err = DB.First(&post, "id = ?", pid).Error
	return &post, err
}
//查询存档的文章 就是按照月份统计
func MustListPostArchives() []*QrArchive {
	archives, _ := ListPostArchives()
	return archives
}

func ListPostArchives() ([]*QrArchive, error) {
	var archives []*QrArchive
	//querysql := `select DATE_FORMAT(created_at,'%Y-%m') as month,count(*) as total from posts where is_published = ? group by month order by month desc`
	querysql := `select strftime('%Y-%m',created_at) as month,count(*) as total from posts where is_published = ? group by month order by month desc`
	rows, err := DB.Raw(querysql, true).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var archive QrArchive
		var month string
		rows.Scan(&month, &archive.Total)
		//DB.ScanRows(rows, &archive)
		archive.ArchiveDate, _ = time.Parse("2006-01", month) //将month格式成年月形式
		archive.Year = archive.ArchiveDate.Year() //获取年
		archive.Month = int(archive.ArchiveDate.Month()) //获取月
		archives = append(archives, &archive)
	}
	return archives, nil
}

func ListPostByArchive(year, month string) ([]*Post, error) {
	if len(month) == 1 {
		month = "0" + month
	}
	condition := fmt.Sprintf("%s-%s", year, month)
	//querysql := `select * from posts where date_format(created_at,'%Y-%m') = ? and is_published = ? order by created_at desc`
	querysql := `select * from posts where strftime('%Y-%m',created_at) = ? and is_published = ? order by created_at desc`
	rows, err := DB.Raw(querysql, condition, true).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	posts := make([]*Post, 0)
	for rows.Next() {
		var post Post
		DB.ScanRows(rows, &post)
		posts = append(posts, &post)
	}
	return posts, nil
}

// Tag
func (tag *Tag) Insert() error {
	//查找或创建一个记录
	return DB.FirstOrCreate(tag, "name = ?", tag.Name).Error
}

func ListTag() ([]*Tag, error) {
	var tags []*Tag
	//查询 tags post_tags posts表 其中post_tags表示tags表和posts表的对照关系表 按照post_tags的tag_id分组统计查询所有已发布
	rows, err := DB.Raw("select t.*,count(*) total from tags t inner join post_tags pt on t.id = pt.tag_id inner join posts p on pt.post_id = p.id where p.is_published = ? group by pt.tag_id", true).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tag Tag
		DB.ScanRows(rows, &tag)
		tags = append(tags, &tag)
	}
	return tags, nil
}

func MustListTag() []*Tag {
	tags, _ := ListTag()
	return tags
}

func ListTagByPostId(id string) ([]*Tag, error) {
	var tags []*Tag
	pid, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, err
	}
	rows, err := DB.Raw("select t.* from tags t inner join post_tags pt on t.id = pt.tag_id where pt.post_id = ?", uint(pid)).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tag Tag
		DB.ScanRows(rows, &tag)
		tags = append(tags, &tag)
	}
	return tags, nil
}

func CountTag() int {
	var count int
	DB.Model(&Tag{}).Count(&count)
	return count
}

func ListAllTag() ([]*Tag, error) {
	var tags []*Tag
	err := DB.Model(&Tag{}).Find(&tags).Error
	return tags, err
}

// post_tags
func (pt *PostTag) Insert() error {
	return DB.FirstOrCreate(pt, "post_id = ? and tag_id = ?", pt.PostId, pt.TagId).Error
}

func DeletePostTagByPostId(postId uint) error {
	return DB.Delete(&PostTag{}, "post_id = ?", postId).Error
}

// user
// insert user
func (user *User) Insert() error {
	return DB.Create(user).Error
}

// update user
func (user *User) Update() error {
	return DB.Save(user).Error
}

//
func GetUserByUsername(username string) (*User, error) {
	var user User
	err := DB.First(&user, "email = ?", username).Error
	return &user, err
}

//获取第一个给定的记录或者创建一个已给定数据的记录
func (user *User) FirstOrCreate() (*User, error) {
	err := DB.FirstOrCreate(user, "github_login_id = ?", user.GithubLoginId).Error
	return user, err
}

func IsGithubIdExists(githubId string, id uint) (*User, error) {
	var user User
	err := DB.First(&user, "github_login_id = ? and id != ?", githubId, id).Error
	return &user, err
}

func GetUser(id interface{}) (*User, error) {
	var user User
	err := DB.First(&user, id).Error
	return &user, err
}

func (user *User) UpdateProfile(avatarUrl, nickName string) error {
	//更新users表中的数据
	return DB.Model(user).Update(User{AvatarUrl: avatarUrl, NickName: nickName}).Error
}

func (user *User) UpdateEmail(email string) error {
	if len(email) > 0 {
		return DB.Model(user).Update("email", email).Error
	} else {
		return DB.Model(user).Update("email", gorm.Expr("NULL")).Error
	}
}

func (user *User) UpdateGithubUserInfo() error {
	var githubLoginId interface{}
	if len(user.GithubLoginId) == 0 {
		githubLoginId = gorm.Expr("NULL")
	} else {
		githubLoginId = user.GithubLoginId
	}
	return DB.Model(user).Update(map[string]interface{}{
		"github_login_id": githubLoginId,
		"avatar_url":      user.AvatarUrl,
		"github_url":      user.GithubUrl,
	}).Error
}

func (user *User) Lock() error {
	return DB.Model(user).Update(map[string]interface{}{
		"lock_state": user.LockState,
	}).Error
}

func ListUsers() ([]*User, error) {
	var users []*User
	err := DB.Find(&users, "is_admin = ?", false).Error
	return users, err
}

// Comment 文章评论
func (comment *Comment) Insert() error {
	return DB.Create(comment).Error
}

func (comment *Comment) Update() error {
	return DB.Model(comment).UpdateColumn("read_state", true).Error
}

func SetAllCommentRead() error {
	return DB.Model(&Comment{}).Where("read_state = ?", false).Update("read_state", true).Error
}

func ListUnreadComment() ([]*Comment, error) {
	var comments []*Comment
	err := DB.Where("read_state = ?", false).Order("created_at desc").Find(&comments).Error
	return comments, err
}

func MustListUnreadComment() []*Comment {
	comments, _ := ListUnreadComment()
	return comments
}

func (comment *Comment) Delete() error {
	return DB.Delete(comment, "user_id = ?", comment.UserID).Error
}

func ListCommentByPostID(postId string) ([]*Comment, error) {
	pid, err := strconv.ParseUint(postId, 10, 64)
	if err != nil {
		return nil, err
	}
	var comments []*Comment
	rows, err := DB.Raw("select c.*,u.github_login_id nick_name,u.avatar_url,u.github_url from comments c inner join users u on c.user_id = u.id where c.post_id = ? order by created_at desc", uint(pid)).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var comment Comment
		DB.ScanRows(rows, &comment)
		comments = append(comments, &comment)
	}
	return comments, err
}

/*func GetComment(id interface{}) (*Comment, error) {
	var comment Comment
	err := DB.First(&comment, id).Error
	return &comment, err
}*/

func CountComment() int {
	var count int
	DB.Model(&Comment{}).Count(&count)
	return count
}

// Subscriber
func (s *Subscriber) Insert() error {
	return DB.FirstOrCreate(s, "email = ?", s.Email).Error
}

func (s *Subscriber) Update() error {
	return DB.Model(s).Update(map[string]interface{}{
		"verify_state":    s.VerifyState,
		"subscribe_state": s.SubscribeState,
		"out_time":        s.OutTime,
		"signature":       s.Signature,
		"secret_key":      s.SecretKey,
	}).Error
}

func ListSubscriber(invalid bool) ([]*Subscriber, error) {
	var subscribers []*Subscriber
	db := DB.Model(&Subscriber{})
	if invalid {
		db.Where("verify_state = ? and subscribe_state = ?", true, true)
	}
	err := db.Find(&subscribers).Error
	return subscribers, err
}
//获取所有有效订阅统计
func CountSubscriber() (int, error) {
	var count int
	err := DB.Model(&Subscriber{}).Where("verify_state = ? and subscribe_state = ?", true, true).Count(&count).Error
	return count, err
}
//查询email下的订阅，email是唯一索引，所以匹配结果最多一个
func GetSubscriberByEmail(mail string) (*Subscriber, error) {
	var subscriber Subscriber
	err := DB.Find(&subscriber, "email = ?", mail).Error
	return &subscriber, err
}

func GetSubscriberBySignature(key string) (*Subscriber, error) {
	var subscriber Subscriber
	err := DB.Find(&subscriber, "signature = ?", key).Error
	return &subscriber, err
}

func GetSubscriberById(id uint) (*Subscriber, error) {
	var subscriber Subscriber
	err := DB.First(&subscriber, id).Error
	return &subscriber, err
}

// Link
func (link *Link) Insert() error {
	return DB.FirstOrCreate(link, "url = ?", link.Url).Error
}

func (link *Link) Update() error {
	return DB.Save(link).Error
}

func (link *Link) Delete() error {
	return DB.Delete(link).Error
}

func ListLinks() ([]*Link, error) {
	var links []*Link
	//查询所有links表 并按照sort升序排列
	err := DB.Order("sort asc").Find(&links).Error
	return links, err
}

func MustListLinks() []*Link {
	links, _ := ListLinks()
	return links
}

func GetLinkById(id uint) (*Link, error) {
	var link Link
	err := DB.FirstOrCreate(&link, "id = ?", id).Error
	return &link, err
}

/*func GetLinkByUrl(url string) (*Link, error) {
	var link Link
	err := DB.Find(&link, "url = ?", url).Error
	return &link, err
}*/
