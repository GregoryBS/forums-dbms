package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aerogo/aero"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	timeLayout = "2006-01-02T15:04:05.000Z"
)

type DBManager struct {
	Pool *pgxpool.Pool
}

type Usecase struct {
	db *DBManager
}

type Handler struct {
	u *Usecase
}

type JSONError struct {
	Message string `json:"message,omitempty"`
}

type DBStatus struct {
	User   int `json:"user"`
	Forum  int `json:"forum"`
	Thread int `json:"thread"`
	Post   int `json:"post"`
}

type User struct {
	Nick     string `json:"nickname,omitempty"`
	Fullname string `json:"fullname,omitempty"`
	Email    string `json:"email,omitempty"`
	About    string `json:"about,omitempty"`
}

type Forum struct {
	Slug    string `json:"slug,omitempty"`
	Title   string `json:"title,omitempty"`
	User    string `json:"user,omitempty"`
	Threads int    `json:"threads,omitempty"`
	Posts   int    `json:"posts,omitempty"`
}

type Thread struct {
	ID      int       `json:"id,omitempty"`
	Title   string    `json:"title,omitempty"`
	Author  string    `json:"author,omitempty"`
	Forum   string    `json:"forum,omitempty"`
	Message string    `json:"message,omitempty"`
	Votes   int       `json:"votes,omitempty"`
	Slug    string    `json:"slug,omitempty"`
	Created time.Time `json:"created,omitempty"`
}

type Post struct {
	ID       int       `json:"id,omitempty"`
	Parent   int       `json:"parent,omitempty"`
	Author   string    `json:"author,omitempty"`
	Forum    string    `json:"forum,omitempty"`
	Thread   int       `json:"thread,omitempty"`
	Message  string    `json:"message,omitempty"`
	IsEdited bool      `json:"isEdited"`
	Created  time.Time `json:"created,omitempty"`
}

type PostForm struct {
	Parent  int    `json:"parent,omitempty"`
	Author  string `json:"author"`
	Message string `json:"message"`
	Path    []int  `json:"-"`
}

type PostDetail struct {
	Details  *Post   `json:"post"`
	Author   *User   `json:"author,omitempty"`
	ForumIn  *Forum  `json:"forum,omitempty"`
	ThreadIn *Thread `json:"thread,omitempty"`
}

type Vote struct {
	Voice  int    `json:"voice"`
	Author string `json:"nickname"`
}

type Counter struct {
	sync.Mutex
	count int
}

var (
	timeNull = time.Time{}
	stat     = make([]Counter, 4)
)

func DecodeJSON(body io.Reader, dst interface{}) error {
	return json.NewDecoder(body).Decode(dst)
}

func main() {
	app := aero.New()
	db := ConnectDB()
	usecases := &Usecase{db}
	handlers := &Handler{usecases}

	app.OnEnd(func() {
		DisconnectDB(db)
	})

	// counter := 0
	// go func() {
	// 	for {
	// 		time.Sleep(time.Second)
	// 		counter += 1
	// 		fmt.Println(counter)
	// 	}
	// }()

	// app.Use(func(next aero.Handler) aero.Handler {
	// 	return func(ctx aero.Context) error {
	// 		start := time.Now()
	// 		err := next(ctx)
	// 		fmt.Println(ctx.Path(), err, time.Since(start))

	// 		return err
	// 	}
	// })

	app = configure(app, handlers)
	app.Run()
}

func ConnectDB() *DBManager {
	config, _ := pgxpool.ParseConfig("host=localhost port=5432 database=forums user=anna password=yoh")
	config.MinConns = 50
	config.MaxConns = 50
	config.ConnConfig.PreferSimpleProtocol = true
	config.ConnConfig.RuntimeParams = map[string]string{
		"standard_conforming_strings": "on",
	}
	pool, err := pgxpool.ConnectConfig(context.Background(), config)
	if err == nil {
		return &DBManager{pool}
	}
	return nil
}

func DisconnectDB(db *DBManager) {
	db.Pool.Close()
}

func configure(app *aero.Application, handlers *Handler) *aero.Application {
	app.Post("/api/user/:nick/create", handlers.SignUp)
	app.Get("/api/user/:nick/profile", handlers.Profile)
	app.Post("/api/user/:nick/profile", handlers.UpdateProfile)
	app.Post("/api/forum/create", handlers.CreateForum)
	app.Get("/api/forum/:slug/details", handlers.GetForum)
	app.Post("/api/forum/:slug/create", handlers.CreateThread)
	app.Post("/api/thread/:slug_or_id/create", handlers.CreatePosts)
	app.Get("/api/thread/:slug_or_id/details", handlers.GetThread)
	app.Get("/api/forum/:slug/threads", handlers.ForumThreads)
	app.Post("/api/service/clear", handlers.Clear)
	app.Get("/api/service/status", handlers.Status)
	app.Post("/api/thread/:slug_or_id/vote", handlers.ThreadVote)
	app.Post("/api/thread/:slug_or_id/details", handlers.UpdateThread)
	app.Get("/api/thread/:slug_or_id/posts", handlers.ThreadPosts)
	app.Get("/api/forum/:slug/users", handlers.ForumUsers)
	app.Post("/api/post/:id/details", handlers.UpdatePost)
	app.Get("/api/post/:id/details", handlers.GetPost)
	return app
}

func (h *Handler) SignUp(ctx aero.Context) error {
	user := &User{}
	if err := DecodeJSON(ctx.Request().Body().Reader(), user); err != nil {
		return ctx.Error(http.StatusInternalServerError)
	}
	user.Nick = ctx.Get("nick")
	response, status := h.u.SignUp(user)
	ctx.SetStatus(status)
	if status == http.StatusCreated {
		stat[3].Lock()
		stat[3].count += 1
		stat[3].Unlock()
		return ctx.JSON(response[0])
	}
	return ctx.JSON(response)
}

func (h *Handler) Profile(ctx aero.Context) error {
	nick := ctx.Get("nick")
	response, status := h.u.Profile(nick)
	ctx.SetStatus(status)
	if status == http.StatusOK {
		return ctx.JSON(response)
	}
	return ctx.JSON(&JSONError{"user not found"})
}

func (h *Handler) UpdateProfile(ctx aero.Context) error {
	user := &User{}
	if err := DecodeJSON(ctx.Request().Body().Reader(), user); err != nil {
		return ctx.Error(http.StatusInternalServerError)
	}
	user.Nick = ctx.Get("nick")
	response, status := h.u.UpdateProfile(user)
	ctx.SetStatus(status)
	switch status {
	case http.StatusOK:
		return ctx.JSON(response)
	case http.StatusNotFound:
		return ctx.JSON(&JSONError{"user not found"})
	case http.StatusConflict:
		return ctx.JSON(&JSONError{"cannot update user"})
	default:
		return ctx.Error(http.StatusInternalServerError)
	}
}

func (h *Handler) CreateForum(ctx aero.Context) error {
	forum := &Forum{}
	if err := DecodeJSON(ctx.Request().Body().Reader(), forum); err != nil {
		return ctx.Error(http.StatusInternalServerError)
	}
	response, status := h.u.CreateForum(forum)
	ctx.SetStatus(status)
	if status == http.StatusNotFound {
		return ctx.JSON(&JSONError{"user not found"})
	}
	stat[0].Lock()
	stat[0].count += 1
	stat[0].Unlock()
	return ctx.JSON(response)
}

func (h *Handler) GetForum(ctx aero.Context) error {
	slug := ctx.Get("slug")
	response, status := h.u.GetForum(slug)
	ctx.SetStatus(status)
	if status == http.StatusOK {
		return ctx.JSON(response)
	}
	return ctx.JSON(&JSONError{"forum not found"})
}

func (h *Handler) CreateThread(ctx aero.Context) error {
	thread := &Thread{}
	if err := DecodeJSON(ctx.Request().Body().Reader(), thread); err != nil {
		return ctx.Error(http.StatusInternalServerError)
	}
	thread.Forum = ctx.Get("slug")
	response, status := h.u.CreateThread(thread)
	ctx.SetStatus(status)
	if status == http.StatusNotFound {
		return ctx.JSON(&JSONError{"user or forum not found"})
	}
	stat[2].Lock()
	stat[2].count += 1
	stat[2].Unlock()
	return ctx.JSON(response)
}

func (h *Handler) CreatePosts(ctx aero.Context) error {
	posts := make([]*PostForm, 0)
	if err := DecodeJSON(ctx.Request().Body().Reader(), &posts); err != nil {
		return ctx.Error(http.StatusInternalServerError)
	}
	slugID := ctx.Get("slug_or_id")
	id, err := strconv.Atoi(slugID)
	var status int
	response := make([]*Post, 0)
	if err == nil {
		response, status = h.u.CreatePosts(posts, id, true)
	} else {
		response, status = h.u.CreatePosts(posts, slugID, false)
	}
	ctx.SetStatus(status)
	switch status {
	case http.StatusCreated:
		stat[1].Lock()
		stat[1].count += len(posts)
		stat[1].Unlock()
		return ctx.JSON(response)
	case http.StatusNotFound:
		return ctx.JSON(&JSONError{"user or thread not found"})
	case http.StatusConflict:
		return ctx.JSON(&JSONError{"cannot create posts"})
	default:
		return ctx.Error(http.StatusInternalServerError)
	}
}

func (h *Handler) GetThread(ctx aero.Context) error {
	slugID := ctx.Get("slug_or_id")
	id, err := strconv.Atoi(slugID)
	var thread *Thread
	var status int
	if err == nil {
		thread, status = h.u.GetThread(id, true)
	} else {
		thread, status = h.u.GetThread(slugID, false)
	}
	ctx.SetStatus(status)
	if status == http.StatusOK {
		return ctx.JSON(thread)
	}
	return ctx.JSON(&JSONError{"thread not found"})
}

func (h *Handler) ForumThreads(ctx aero.Context) error {
	slug := ctx.Get("slug")
	url := ctx.Request().Internal().URL
	limitParam := url.Query().Get("limit")
	sinceParam := url.Query().Get("since")
	desc := url.Query().Get("desc")
	limit, err := strconv.Atoi(limitParam)
	if err != nil {
		limit = 100
	}
	since, err := time.Parse(timeLayout, sinceParam)
	if err != nil {
		since = timeNull
	}
	response, status := h.u.ForumThreads(slug, limit, since, desc)
	ctx.SetStatus(status)
	if status == http.StatusOK {
		return ctx.JSON(response)
	}
	return ctx.JSON(&JSONError{"forum not found"})
}

func (h *Handler) Clear(ctx aero.Context) error {
	status := h.u.Clear()
	return ctx.Error(status)
}

func (h *Handler) Status(ctx aero.Context) error {
	response, status := h.u.Status()
	ctx.SetStatus(status)
	return ctx.JSON(response)
}

func (h *Handler) ThreadVote(ctx aero.Context) error {
	vote := &Vote{}
	if err := DecodeJSON(ctx.Request().Body().Reader(), vote); err != nil {
		return ctx.Error(http.StatusInternalServerError)
	}
	slugID := ctx.Get("slug_or_id")
	id, err := strconv.Atoi(slugID)
	var thread *Thread
	var status int
	if err == nil {
		thread, status = h.u.ThreadVote(vote, id, true)
	} else {
		thread, status = h.u.ThreadVote(vote, slugID, false)
	}
	ctx.SetStatus(status)
	if status == http.StatusOK {
		return ctx.JSON(thread)
	}
	return ctx.JSON(&JSONError{"user or thread not found"})
}

func (h *Handler) UpdateThread(ctx aero.Context) error {
	thread := &Thread{}
	if err := DecodeJSON(ctx.Request().Body().Reader(), thread); err != nil {
		return ctx.Error(http.StatusInternalServerError)
	}
	slugID := ctx.Get("slug_or_id")
	id, err := strconv.Atoi(slugID)
	var status int
	if err == nil {
		thread, status = h.u.UpdateThread(thread, id, true)
	} else {
		thread, status = h.u.UpdateThread(thread, slugID, false)
	}
	ctx.SetStatus(status)
	switch status {
	case http.StatusOK:
		return ctx.JSON(thread)
	case http.StatusNotFound:
		return ctx.JSON(&JSONError{"thread not found"})
	case http.StatusConflict:
		return ctx.JSON(&JSONError{"cannot update thread"})
	default:
		return ctx.Error(http.StatusInternalServerError)
	}
}

func (h *Handler) ThreadPosts(ctx aero.Context) error {
	url := ctx.Request().Internal().URL
	limitParam := url.Query().Get("limit")
	sinceParam := url.Query().Get("since")
	sort := url.Query().Get("sort")
	desc := url.Query().Get("desc")
	limit, err := strconv.Atoi(limitParam)
	if err != nil {
		limit = 100
	}
	since, err := strconv.Atoi(sinceParam)
	if err != nil {
		since = 0
	}
	if sort == "" {
		sort = "flat"
	}
	slugID := ctx.Get("slug_or_id")
	id, err := strconv.Atoi(slugID)
	var response []*Post
	var status int
	if err == nil {
		response, status = h.u.ThreadPosts(id, true, limit, since, sort, desc)
	} else {
		response, status = h.u.ThreadPosts(slugID, false, limit, since, sort, desc)
	}
	ctx.SetStatus(status)
	if status == http.StatusOK {
		return ctx.JSON(response)
	}
	return ctx.JSON(&JSONError{"thread not found"})
}

func (h *Handler) ForumUsers(ctx aero.Context) error {
	slug := ctx.Get("slug")
	url := ctx.Request().Internal().URL
	limitParam := url.Query().Get("limit")
	since := url.Query().Get("since")
	desc := url.Query().Get("desc")
	limit, err := strconv.Atoi(limitParam)
	if err != nil {
		limit = 100
	}
	response, status := h.u.ForumUsers(slug, limit, since, desc)
	ctx.SetStatus(status)
	if status == http.StatusOK {
		return ctx.JSON(response)
	}
	return ctx.JSON(&JSONError{"forum not found"})
}

func (h *Handler) UpdatePost(ctx aero.Context) error {
	post := &PostForm{}
	if err := DecodeJSON(ctx.Request().Body().Reader(), post); err != nil {
		return ctx.Error(http.StatusInternalServerError)
	}
	id, err := strconv.Atoi(ctx.Get("id"))
	if err != nil {
		return ctx.Error(http.StatusInternalServerError)
	}
	response, status := h.u.UpdatePost(post, id)
	ctx.SetStatus(status)
	if status == http.StatusOK {
		return ctx.JSON(response)
	}
	return ctx.JSON(&JSONError{"post not found"})
}

func (h *Handler) GetPost(ctx aero.Context) error {
	id, err := strconv.Atoi(ctx.Get("id"))
	if err != nil {
		return ctx.Error(http.StatusInternalServerError)
	}
	related := ctx.Request().Internal().URL.Query().Get("related")
	response, status := h.u.GetPost(id, related)
	ctx.SetStatus(status)
	if status == http.StatusOK {
		return ctx.JSON(response)
	}
	return ctx.JSON(&JSONError{"post not found"})
}

func (u *Usecase) SignUp(user *User) ([]*User, int) {
	ctx := context.Background()
	result := make([]*User, 0)
	tx, err := u.db.Pool.Begin(ctx)
	if err != nil {
		return nil, 500
	}
	_, err = tx.Exec(ctx, "insert into users values($1, $2, $3, $4);", user.Nick, user.Fullname, user.Email, user.About)
	if err != nil {
		tx.Rollback(ctx)
		tx, err = u.db.Pool.Begin(ctx)
		if err != nil {
			return nil, 500
		}
		rows, err := tx.Query(ctx, "select nickname, fullname, email, about from users where nickname = $1 or email = $2;", user.Nick, user.Email)
		if err != nil {
			tx.Rollback(ctx)
			fmt.Println("creating user error: ", err)
			return nil, 500
		}
		defer rows.Close()

		for rows.Next() {
			var nick, full, email, about string
			err = rows.Scan(&nick, &full, &email, &about)
			if err != nil {
				fmt.Println(err)
				return nil, 500
			}
			result = append(result, &User{nick, full, email, about})
		}
		tx.Commit(ctx)
		return result, 409
	}
	tx.Commit(ctx)
	result = append(result, user)
	return result, 201
}

func (u *Usecase) Profile(nick string) (*User, int) {
	user := &User{}
	ctx := context.Background()
	tx, err := u.db.Pool.Begin(ctx)
	if err != nil {
		return nil, 500
	}
	row := tx.QueryRow(context.Background(), "select nickname, fullname, email, about from users where nickname = $1;", nick)
	err = row.Scan(&user.Nick, &user.Fullname, &user.Email, &user.About)
	if err == nil {
		tx.Commit(ctx)
		return user, 200
	}
	tx.Rollback(ctx)
	return nil, 404
}

func (u *Usecase) UpdateProfile(user *User) (*User, int) {
	counter := 1
	fields := make([]interface{}, 0)
	query := "update users set "
	if user.Fullname != "" {
		query += fmt.Sprintf("fullname = $%d", counter)
		fields = append(fields, user.Fullname)
		counter += 1
	}
	if user.Email != "" {
		if counter > 1 {
			query += ","
		}
		query += fmt.Sprintf("email = $%d", counter)
		fields = append(fields, user.Email)
		counter += 1
	}
	if user.About != "" {
		if counter > 1 {
			query += ","
		}
		query += fmt.Sprintf("about = $%d", counter)
		fields = append(fields, user.About)
		counter += 1
	}
	if counter == 1 {
		return u.Profile(user.Nick)
	}
	query += fmt.Sprintf(" where nickname = $%d returning *;", counter)
	fields = append(fields, user.Nick)

	ctx := context.Background()
	tx, err := u.db.Pool.Begin(ctx)
	if err != nil {
		return nil, 500
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(context.Background(), query, fields...)
	var nick, full, email, about string
	err = row.Scan(&nick, &full, &email, &about)
	if err == nil {
		tx.Commit(ctx)
		return &User{nick, full, email, about}, 200
	} else if err == pgx.ErrNoRows {
		return nil, 404
	}
	return nil, 409
}

func (u *Usecase) CreateForum(forum *Forum) (*Forum, int) {
	ctx := context.Background()
	tx, err := u.db.Pool.Begin(ctx)
	if err != nil {
		return nil, 500
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, "select nickname from users where nickname = $1;", forum.User)
	if err := row.Scan(&forum.User); err != nil {
		return nil, 404
	}

	row = u.db.Pool.QueryRow(ctx, "insert into forums values($1, $2, $3, 0, 0) returning *;", forum.Slug, forum.Title, forum.User)
	if err := row.Scan(&forum.Slug, &forum.Title, &forum.User, &forum.Threads, &forum.Posts); err != nil {
		forum, _ = u.GetForum(forum.Slug)
		return forum, 409
	}
	tx.Commit(ctx)
	return forum, 201
}

func (u *Usecase) GetForum(slug string) (*Forum, int) {
	ctx := context.Background()
	tx, err := u.db.Pool.Begin(ctx)
	if err != nil {
		return nil, 500
	}
	defer tx.Rollback(ctx)

	forum := &Forum{}
	row := tx.QueryRow(context.Background(), "select slug, title, author, threads, posts from forums where slug = $1;", slug)
	if err := row.Scan(&forum.Slug, &forum.Title, &forum.User, &forum.Threads, &forum.Posts); err != nil {
		return nil, 404
	}
	tx.Commit(ctx)
	return forum, 200
}

func (u *Usecase) CreateThread(thread *Thread) (*Thread, int) {
	ctx := context.Background()
	tx, err := u.db.Pool.Begin(ctx)
	if err != nil {
		return nil, 500
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, "select nickname as slug from users where nickname = $1 union all select slug from forums where slug = $2;",
		thread.Author, thread.Forum)
	if err != nil {
		return nil, 404
	}
	defer rows.Close()
	keys := make([]string, 0)
	for rows.Next() {
		var buf string
		err = rows.Scan(&buf)
		if err != nil {
			return nil, 500
		}
		keys = append(keys, buf)
	}
	if len(keys) < 2 {
		return nil, 404
	}

	if thread.Created == timeNull {
		thread.Created = time.Now()
	}
	thread.Created = thread.Created.Round(time.Microsecond)
	row := u.db.Pool.QueryRow(ctx, "insert into threads values(default, $1, $2, $3, $4, nullif($5, ''), $6, 0) returning id, author, forum;",
		thread.Title, keys[0], keys[1], thread.Message, thread.Slug, thread.Created)
	err = row.Scan(&thread.ID, &thread.Author, &thread.Forum)
	if err != nil {
		thread, _ = u.GetThread(thread.Slug, false)
		return thread, 409
	}
	tx.Commit(ctx)
	return thread, 201
}

func (u *Usecase) CreatePosts(posts []*PostForm, slugID interface{}, flag bool) ([]*Post, int) {
	ctx := context.Background()
	var row pgx.Row
	var id int
	var forum string
	if flag {
		row = u.db.Pool.QueryRow(ctx, "select id, forum from threads where id = $1;", slugID)
	} else {
		row = u.db.Pool.QueryRow(ctx, "select id, forum from threads where slug = $1;", slugID)
	}
	if err := row.Scan(&id, &forum); err != nil {
		return nil, 404
	}

	conn, err := u.db.Pool.Acquire(ctx)
	if err != nil {
		return nil, 500
	}
	defer conn.Release()

	_, err = conn.Conn().Prepare(ctx, "post_path", "select thread, path from posts where id = $1;")
	if err != nil {
		return nil, 500
	}

	for i := range posts {
		if posts[i].Parent != 0 {
			var threadID int
			path := make([]int, 0)
			row = conn.Conn().QueryRow(ctx, "post_path", posts[i].Parent)
			if err := row.Scan(&threadID, &path); err != nil || threadID != id {
				return nil, 409
			}
			posts[i].Path = path
		}
	}

	query := "insert into posts values (default, $1, $2, $3, $4, $5, $6, false, $7)"
	for i := 1; i < len(posts); i += 1 {
		query += fmt.Sprintf(",(default, $%d, $%d, $%d, $%d, $%d, $%d, false, $%d)", i*7+1, i*7+2, i*7+3, i*7+4, i*7+5, i*7+6, i*7+7)
	}
	query += " returning id, parent, author, message, edit;"

	created := time.Now().Round(time.Microsecond)
	fields := make([]interface{}, 0)
	for i := range posts {
		fields = append(fields, posts[i].Parent, posts[i].Author, forum, id, posts[i].Message, created, posts[i].Path)
	}

	result := make([]*Post, 0)
	if len(fields) > 0 {
		rows, err := conn.Conn().Query(ctx, query, fields...)
		if err == pgx.ErrNoRows {
			return nil, 409
		} else if err != nil {
			return nil, 404
		}
		defer rows.Close()

		for rows.Next() {
			post := &Post{}
			err = rows.Scan(&post.ID, &post.Parent, &post.Author, &post.Message, &post.IsEdited)
			if err != nil {
				return nil, 500
			}
			post.Created = created
			post.Forum = forum
			post.Thread = id
			result = append(result, post)
		}
		conn.Conn().Exec(ctx, fmt.Sprintf("update forums set posts = posts + %d where slug = $1;", len(posts)), forum)

		fields = make([]interface{}, 1)
		fields[0] = forum
		query = "insert into forum_users select $1, nickname, fullname, email, about from users where nickname in ("
		for i := range result {
			query += fmt.Sprintf("$%d,", i+2)
			fields = append(fields, result[i].Author)
		}
		query = strings.TrimSuffix(query, ",")
		query += ") on conflict do nothing;"
		conn.Conn().Exec(ctx, query, fields...)
	}
	return result, 201
}

func (u *Usecase) GetThread(slugID interface{}, flag bool) (*Thread, int) {
	ctx := context.Background()
	tx, err := u.db.Pool.Begin(ctx)
	if err != nil {
		return nil, 500
	}
	defer tx.Rollback(ctx)

	var row pgx.Row
	if flag {
		row = tx.QueryRow(ctx, "select id, forum, title, author, created, message, slug, votes from threads where id = $1;", slugID)
	} else {
		row = tx.QueryRow(ctx, "select id, forum, title, author, created, message, slug, votes from threads where slug = $1;", slugID)
	}
	thread := &Thread{}
	var slug sql.NullString
	err = row.Scan(&thread.ID, &thread.Forum, &thread.Title, &thread.Author, &thread.Created, &thread.Message, &slug, &thread.Votes)
	if err != nil {
		return nil, 404
	}
	thread.Created = thread.Created.Round(time.Microsecond)
	if slug.Valid {
		thread.Slug = slug.String
	}
	tx.Commit(ctx)
	return thread, 200
}

func (u *Usecase) ForumThreads(forum string, limit int, since time.Time, desc string) ([]*Thread, int) {
	counter := 2
	fields := make([]interface{}, 1)
	fields[0] = forum
	query := "select id, forum, title, author, created, message, slug, votes from threads where forum = $1 "
	if since != timeNull {
		if desc == "true" {
			query += fmt.Sprintf("and created <= $%d ", counter)
		} else {
			query += fmt.Sprintf("and created >= $%d ", counter)
		}
		fields = append(fields, since)
		counter += 1
	}

	query += "order by created "
	if desc == "true" {
		query += "desc "
	}
	query += fmt.Sprintf("limit $%d;", counter)
	fields = append(fields, limit)

	ctx := context.Background()
	tx, err := u.db.Pool.Begin(ctx)
	if err != nil {
		return nil, 500
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, "select slug from forums where slug = $1;", forum)
	if err := row.Scan(&forum); err != nil {
		return nil, 404
	}
	rows, err := tx.Query(ctx, query, fields...)
	if err != nil {
		return nil, 500
	}
	defer rows.Close()

	result := make([]*Thread, 0)
	for rows.Next() {
		thread := &Thread{}
		var slug sql.NullString
		err = rows.Scan(&thread.ID, &thread.Forum, &thread.Title, &thread.Author, &thread.Created, &thread.Message, &slug, &thread.Votes)
		if err != nil {
			return nil, 500
		}
		thread.Created = thread.Created.Round(time.Microsecond)
		if slug.Valid {
			thread.Slug = slug.String
		}
		result = append(result, thread)
	}
	tx.Commit(ctx)
	return result, 200
}

func (u *Usecase) Clear() int {
	_, err := u.db.Pool.Exec(context.Background(), "truncate users cascade;")
	if err != nil {
		return 500
	}
	for i := range stat {
		stat[i].Lock()
		stat[i].count = 0
		stat[i].Unlock()
	}
	return 200
}

func (u *Usecase) Status() (*DBStatus, int) {
	for i := range stat {
		stat[i].Lock()
	}
	result := &DBStatus{stat[3].count, stat[0].count, stat[2].count, stat[1].count}
	for i := range stat {
		stat[i].Unlock()
	}
	return result, 200
}

func (u *Usecase) ThreadVote(vote *Vote, slugID interface{}, flag bool) (*Thread, int) {
	ctx := context.Background()
	tx, err := u.db.Pool.Begin(ctx)
	if err != nil {
		return nil, 500
	}
	defer tx.Rollback(ctx)

	var row pgx.Row
	var id int
	if flag {
		row = tx.QueryRow(ctx, "select id from threads where id = $1;", slugID)
	} else {
		row = tx.QueryRow(ctx, "select id from threads where slug = $1;", slugID)
	}
	if err := row.Scan(&id); err != nil {
		return nil, 404
	}

	_, err = tx.Exec(ctx, "insert into votes values($1, $2, $3);", vote.Author, id, vote.Voice)
	if pgerr, ok := err.(*pgconn.PgError); ok {
		if pgerr.Code == "23505" {
			tx.Rollback(ctx)
			tx, err = u.db.Pool.Begin(ctx)
			if err != nil {
				return nil, 500
			}
			tx.Exec(ctx, "update votes set value = $1 where author = $2 and thread = $3;", vote.Voice, vote.Author, id)
		} else {
			return nil, 404
		}
	}
	tx.Commit(ctx)
	return u.GetThread(id, true)
}

func (u *Usecase) UpdateThread(thread *Thread, slugID interface{}, flag bool) (*Thread, int) {
	counter := 1
	fields := make([]interface{}, 0)
	query := "update threads set "
	if thread.Title != "" {
		query += fmt.Sprintf("title = $%d", counter)
		fields = append(fields, thread.Title)
		counter += 1
	}
	if thread.Message != "" {
		if counter > 1 {
			query += ","
		}
		query += fmt.Sprintf("message = $%d", counter)
		fields = append(fields, thread.Message)
		counter += 1
	}
	if counter == 1 {
		return u.GetThread(slugID, flag)
	}
	if flag {
		query += fmt.Sprintf(" where id = $%d returning *;", counter)
	} else {
		query += fmt.Sprintf(" where slug = $%d returning *;", counter)
	}
	fields = append(fields, slugID)

	ctx := context.Background()
	tx, err := u.db.Pool.Begin(ctx)
	if err != nil {
		return nil, 500
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(context.Background(), query, fields...)
	var slug sql.NullString
	err = row.Scan(&thread.ID, &thread.Title, &thread.Author, &thread.Forum, &thread.Message, &slug, &thread.Created, &thread.Votes)
	if err != nil {
		return nil, 404
	}
	thread.Created = thread.Created.Round(time.Microsecond)
	if slug.Valid {
		thread.Slug = slug.String
	}
	tx.Commit(ctx)
	return thread, 200
}

func (u *Usecase) ThreadPosts(slugID interface{}, flag bool, limit, since int, sort, desc string) ([]*Post, int) {
	ctx := context.Background()
	tx, err := u.db.Pool.Begin(ctx)
	if err != nil {
		return nil, 500
	}
	defer tx.Rollback(ctx)

	var row pgx.Row
	var id int
	if flag {
		row = tx.QueryRow(ctx, "select id from threads where id = $1;", slugID)
	} else {
		row = tx.QueryRow(ctx, "select id from threads where slug = $1;", slugID)
	}
	if err := row.Scan(&id); err != nil {
		return nil, 404
	}

	counter := 2
	fields := make([]interface{}, 1)
	fields[0] = id
	query := "select id, parent, author, forum, thread, message, created, edit from posts where thread = $1 "
	switch sort {
	case "flat":
		if since > 0 {
			if desc == "true" {
				query += fmt.Sprintf("and id < $%d ", counter)
			} else {
				query += fmt.Sprintf("and id > $%d ", counter)
			}
			fields = append(fields, since)
			counter += 1
		}
		if desc == "true" {
			query += "order by created desc, id desc "
		} else {
			query += "order by created, id "
		}
		query += fmt.Sprintf("limit $%d;", counter)
	case "tree":
		if since > 0 {
			if desc == "true" {
				query += fmt.Sprintf("and path < (select path from posts where id = $%d) ", counter)
			} else {
				query += fmt.Sprintf("and path > (select path from posts where id = $%d) ", counter)
			}
			fields = append(fields, since)
			counter += 1
		}
		query += "order by path "
		if desc == "true" {
			query += "desc "
		}
		query += fmt.Sprintf("limit $%d;", counter)
	case "parent_tree":
		query += "and path[1] in (select id from posts where thread = $1 and parent = 0 "
		if since > 0 {
			if desc == "true" {
				query += fmt.Sprintf("and path[1] < (select path[1] from posts where id = $%d) ", counter)
			} else {
				query += fmt.Sprintf("and path[1] > (select path[1] from posts where id = $%d) ", counter)
			}
			fields = append(fields, since)
			counter += 1
		}
		if desc == "true" {
			query += fmt.Sprintf("order by id desc limit $%d) order by path[1] desc, path;", counter)
		} else {
			query += fmt.Sprintf("order by id limit $%d) order by path[1], path;", counter)
		}
	default:
		return nil, 500
	}
	fields = append(fields, limit)

	rows, err := tx.Query(ctx, query, fields...)
	if err != nil {
		return nil, 500
	}
	defer rows.Close()
	result := make([]*Post, 0)
	for rows.Next() {
		p := &Post{}
		err = rows.Scan(&p.ID, &p.Parent, &p.Author, &p.Forum, &p.Thread, &p.Message, &p.Created, &p.IsEdited)
		if err != nil {
			return nil, 500
		}
		p.Created = p.Created.Round(time.Microsecond)
		result = append(result, p)
	}
	tx.Commit(ctx)
	return result, 200
}

func (u *Usecase) ForumUsers(slug string, limit int, since, desc string) ([]*User, int) {
	query := `select nickname, fullname, email, about from forum_users where forum = $1 `
	counter := 2
	fields := make([]interface{}, 1)
	fields[0] = slug
	if since != "" {
		since = strings.ToLower(since)
		if desc == "true" {
			query += fmt.Sprintf("and lower(nickname) < $%d ", counter)
		} else {
			query += fmt.Sprintf("and lower(nickname) > $%d ", counter)
		}
		fields = append(fields, since)
		counter += 1
	}
	query += "order by lower(nickname) "
	if desc == "true" {
		query += "desc "
	}
	query += fmt.Sprintf("limit $%d;", counter)
	fields = append(fields, limit)

	ctx := context.Background()
	tx, err := u.db.Pool.Begin(ctx)
	if err != nil {
		return nil, 500
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, "select slug from forums where slug = $1;", slug)
	if err := row.Scan(&slug); err != nil {
		return nil, 404
	}

	rows, err := tx.Query(ctx, query, fields...)
	if err != nil {
		return nil, 500
	}
	defer rows.Close()
	result := make([]*User, 0)
	for rows.Next() {
		u := &User{}
		err = rows.Scan(&u.Nick, &u.Fullname, &u.Email, &u.About)
		if err != nil {
			return nil, 500
		}
		result = append(result, u)
	}
	tx.Commit(ctx)
	return result, 200
}

func (u *Usecase) UpdatePost(post *PostForm, id int) (*Post, int) {
	detailed, status := u.GetPost(id, "")
	if status != 200 {
		return nil, status
	}
	if post.Message == "" || detailed.Details.Message == post.Message {
		return detailed.Details, 200
	}

	ctx := context.Background()
	tx, err := u.db.Pool.Begin(ctx)
	if err != nil {
		return nil, 500
	}
	defer tx.Rollback(ctx)

	tx.Exec(ctx, "update posts set message = $1, edit = true where id = $2;", post.Message, id)
	tx.Commit(ctx)
	detailed.Details.Message = post.Message
	detailed.Details.IsEdited = true
	return detailed.Details, 200
}

func (u *Usecase) GetPost(id int, related string) (*PostDetail, int) {
	post := &Post{}
	ctx := context.Background()
	tx, err := u.db.Pool.Begin(ctx)
	if err != nil {
		return nil, 500
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, "select id, parent, author, forum, thread, message, created, edit from posts where id = $1", id)
	err = row.Scan(&post.ID, &post.Parent, &post.Author, &post.Forum, &post.Thread,
		&post.Message, &post.Created, &post.IsEdited)
	if err != nil {
		return nil, 404
	}
	tx.Commit(ctx)
	post.Created = post.Created.Round(time.Microsecond)

	var forum *Forum
	if strings.Contains(related, "forum") {
		forum, _ = u.GetForum(post.Forum)
	}
	var thread *Thread
	if strings.Contains(related, "thread") {
		thread, _ = u.GetThread(post.Thread, true)
	}
	var author *User
	if strings.Contains(related, "user") {
		author, _ = u.Profile(post.Author)
	}
	return &PostDetail{post, author, forum, thread}, 200
}
