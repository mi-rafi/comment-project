package database

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype/zeronull"
	"github.com/jackc/pgx/v5/pgxpool"
	mod "github.com/mi-raf/comment-project/internal/models"
)

const (
	insertPost    = "INSERT INTO Post (author, title, text_p, comm, time_p) VALUES ($1, $2, $3, $4, $5) RETURNING id_p"
	insertComment = "INSERT INTO Comment (id_p, parent, author, text_c, time_c) VALUES ($1, $2, $3, $4, $5) RETURNING id_c"
	searchAllPost = `SELECT id_p, author, title, text_p, comm, time_p 
	FROM Post 
	ORDER BY time_p DESC
	LIMIT $1
	OFFSET $2`
	searchPost = `SELECT  id_p, author, title, text_p, comm, time_p 
	FROM Post 
	WHERE id_p = $1`
	searchPostIsComment  = "SELECT  comm FROM Post WHERE id_p = $1"
	searchCommentAllPost = `WITH RECURSIVE comment_tree AS (
		SELECT
			id_c,
			parent,
			author, 
			text_c,
			time_c,
			1 AS level,
			ARRAY[id_c] AS path
		FROM Comment
		WHERE parent IS NULL
	
		UNION ALL
	
		SELECT
			c.id_c,
			c.parent,
			c.author, 
			c.text_c,
			c.time_c,
			ct.level + 1 AS level,
			ct.path || c.id_c
		FROM Comment c
		INNER JOIN comment_tree ct ON c.parent = ct.id_c
	),
	numbered_comments AS (
		SELECT *,
			   ROW_NUMBER() OVER (ORDER BY path) AS row_num
		FROM comment_tree
	)
	SELECT id_c, parent, author, text_c, time_c, level
	FROM numbered_comments
	WHERE row_num > $1
	ORDER BY row_num
	LIMIT $2;`
)

const (
	STARTCAP = 128
)

type (
	PostRepository interface {
		//Delete(ctx context.Context) error
		Add(ctx context.Context, post *mod.PostDTO) (int64, error)
		GetAll(ctx context.Context, offset int64, limit int) ([]*mod.PostDTO, error)
		Get(ctx context.Context, id int64) (*mod.PostDTO, error)
		IsCommentable(ctx context.Context, id int64) (bool, error)
	}

	CommentRepository interface {
		Add(ctx context.Context, c *mod.CommentDTO) (int64, error)
		GetAllOfPost(ctx context.Context, idPost int64, offset int64, limit int32) ([]*mod.CommentDTO, error)
	}

	PostConfig struct {
		InMemory bool
		DbAddr   string
	}

	CommentConfig struct {
		InMemory    bool
		DbAddr      string
		CommentChan chan<- mod.CommentDTO
	}

	PgPostRepository struct {
		pool *pgxpool.Pool
	}

	PgCommentRepository struct {
		pool *pgxpool.Pool
		ch   chan<- mod.CommentDTO
	}

	InMemoryPostRepository struct {
		posts map[int64]*mod.PostDTO
		m     sync.RWMutex
		idGen int64
	}

	InMemoryCommentRepository struct {
		c        map[int64][]*mod.CommentDTO
		indexMap map[int64]int
		ch       chan<- mod.CommentDTO
		m        sync.RWMutex
		idGen    int64
	}

	InMemCommentNode struct {
		v      mod.CommentDTO
		childs []*InMemCommentNode
	}

	PostRepositoryProvider struct {
	}

	CommentRepositoryProvider struct {
	}
)

func NewPostRepositoryProvider(ctx context.Context, cfg *PostConfig) (PostRepository, func(), error) {
	if cfg.InMemory {
		return NewInMemnoryPostRepository(), nil, nil
	}

	pg, err := pgxpool.New(ctx, cfg.DbAddr)
	if err != nil {
		return nil, nil, err
	}
	err = pg.Ping(ctx)
	if err != nil {
		return nil, nil, err
	}

	return NewPgPostRepository(ctx, pg), pg.Close, nil
}
func NewCommentRepositoryProvider(ctx context.Context, cfg *CommentConfig) (CommentRepository, func(), error) {
	if cfg.InMemory {
		return NewInMemnoryCommentRepository(cfg.CommentChan), nil, nil
	}
	pg, err := pgxpool.New(ctx, cfg.DbAddr)
	if err != nil {
		return nil, nil, err
	}
	err = pg.Ping(ctx)
	if err != nil {
		return nil, nil, err
	}
	return NewPgCommentRepository(ctx, pg, cfg.CommentChan), pg.Close, nil
}

func NewPgPostRepository(ctx context.Context, p *pgxpool.Pool) *PgPostRepository {
	return &PgPostRepository{pool: p}
}

func (r *PgPostRepository) Close(ctx context.Context) error {
	r.pool.Close()
	return nil
}

func (r *PgPostRepository) Add(ctx context.Context, post *mod.PostDTO) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("can't open transaction for add post")
		return -1, err
	}

	defer func() {
		err = tx.Rollback(ctx)
		if !errors.Is(err, pgx.ErrTxClosed) {
			log.Error().Err(err).Msg("Undefinded error in tx")
		}
	}()

	log.Debug().Interface("post", post).Msg("add post")
	var id int64
	err = tx.QueryRow(ctx, insertPost, post.Author, post.Title, post.Text, post.IsCommentable, post.Time).Scan(&id)

	if err != nil {
		log.Error().Err(err).Msg("error insert received")
		return -1, err
	}

	return id, tx.Commit(ctx)
}

func (r *PgPostRepository) GetAll(ctx context.Context, offset int64, limit int) ([]*mod.PostDTO, error) {
	rows, err := r.pool.Query(ctx, searchAllPost, limit, offset)
	if err != nil {
		return nil, err
	}

	posts := make([]*mod.PostDTO, 0)

	for rows.Next() {
		var p mod.PostDTO
		err = rows.Scan(&p.Id, &p.Author, &p.Title, &p.Text, &p.IsCommentable, &p.Time)
		if err != nil {
			return nil, err
		}
		posts = append(posts, &p)
	}

	return posts, nil
}

func (r *PgPostRepository) Get(ctx context.Context, id int64) (*mod.PostDTO, error) {
	row := r.pool.QueryRow(ctx, searchPost, id)
	p := mod.PostDTO{}
	err := row.Scan(&p.Id, &p.Author, &p.Title, &p.Text, &p.IsCommentable, &p.Time)
	if err == pgx.ErrNoRows {
		log.Error().Msg("no post with this id")
		return nil, nil
	}

	return &p, err
}

func (r *PgPostRepository) IsCommentable(ctx context.Context, id int64) (bool, error) {
	var isCom bool
	err := r.pool.QueryRow(ctx, searchPostIsComment, id).Scan(&isCom)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	return isCom, err
}

func NewPgCommentRepository(ctx context.Context, p *pgxpool.Pool, commentChan chan<- mod.CommentDTO) *PgCommentRepository {
	repo := &PgCommentRepository{pool: p, ch: commentChan}
	go repo.Listen(ctx)
	return repo
}

func (r *PgCommentRepository) Close(ctx context.Context) error {
	r.pool.Close()
	return nil
}

func (r *PgCommentRepository) Add(ctx context.Context, c *mod.CommentDTO) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("can't open transaction for add comment")
		return -1, err
	}

	defer func() {
		err = tx.Rollback(ctx)
		if !errors.Is(err, pgx.ErrTxClosed) {
			log.Error().Err(err).Msg("Undefinded error in tx")
		}
	}()

	log.Debug().Interface("comment", c).Msg("add comment")
	var id int64
	err = tx.QueryRow(ctx, insertComment, c.PostId, zeronull.Int8(c.ParentId), c.Author, c.Text, c.Time).Scan(&id)

	if err != nil {
		log.Error().Err(err).Msg("error insert received")
		return -1, err
	}

	c.Id = id
	j, err := json.Marshal(c)
	if err != nil {
		log.Error().Err(err).Interface("comment", c).Msg("can not marshall comment")
		return -1, err
	}

	_, err = r.pool.Exec(ctx, "select pg_notify('comments', $1)", string(j))
	if err != nil {
		log.Error().Err(err).Msg("can not notify clients")
	}
	return id, tx.Commit(ctx)
}

func (r *PgCommentRepository) GetAllOfPost(ctx context.Context, idPost int64, offset int64, limit int) ([]*mod.CommentDTO, error) {
	log.Debug().Interface("post id", idPost).Msg("get comments for post")
	rows, err := r.pool.Query(ctx, searchCommentAllPost, offset, limit)
	if err == pgx.ErrNoRows {
		log.Debug().Msg("GetAll return 0 rows")
		return nil, nil
	}
	if err != nil {
		log.Error().Err(err).Msg("can't return result getAll")
		return nil, err
	}
	comments := make([]*mod.CommentDTO, 0, limit)
	for rows.Next() {
		c := mod.CommentDTO{PostId: idPost}
		var parIdNil zeronull.Int2
		err = rows.Scan(&c.Id, &parIdNil, &c.Author, &c.Text, &c.Time, &c.Level)
		c.ParentId = int64(parIdNil)
		if err != nil {
			return nil, err
		}
		comments = append(comments, &c)
	}
	return comments, nil
}

func (c *PgCommentRepository) Listen(ctx context.Context) {
	conn, err := c.pool.Acquire(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("can not accure conntection")
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, "listen comments")
	if err != nil {
		log.Fatal().Err(err).Msg("can not listen")
	}

	for {
		n, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			log.Error().Err(err).Msg("listen stopped")
			return
		}

		var res mod.CommentDTO
		log.Debug().Uint32("PID", n.PID).Str("channel", n.Channel).Str("payload", n.Payload).Msg("notification received")
		err = json.Unmarshal([]byte(n.Payload), &res)
		if err != nil {
			log.Error().Err(err).Str("payload", n.Payload).Msg("can not unmarshall notification payload")
			continue
		}
		c.ch <- res
	}
}

func NewInMemnoryPostRepository() *InMemoryPostRepository {
	return &InMemoryPostRepository{}
}

func (r *InMemoryPostRepository) Add(ctx context.Context, post *mod.PostDTO) (int64, error) {
	r.m.Lock()
	defer r.m.Unlock()
	post.Id = r.idGen
	r.idGen++
	r.posts[int64(post.Id)] = post
	return post.Id, nil
}

func (r *InMemoryPostRepository) GetAll(ctx context.Context, offset int64, limit int) ([]*mod.PostDTO, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	p := make([]*mod.PostDTO, 0, limit)
	for i := offset; limit > 0; {
		p = append(p, r.posts[i])
		i++
		limit--
	}
	return p, nil
}

func (r *InMemoryPostRepository) Get(ctx context.Context, id int64) (*mod.PostDTO, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	p := r.posts[id]
	return p, nil
}

func (r *InMemoryPostRepository) IsCommentable(ctx context.Context, id int64) (bool, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	isCom := r.posts[id].IsCommentable
	return isCom, nil
}

func NewInMemnoryCommentRepository(ch chan<- mod.CommentDTO) *InMemoryCommentRepository {
	c := make(map[int64][]*mod.CommentDTO, STARTCAP)
	return &InMemoryCommentRepository{c: c, ch: ch, idGen: 1}
}

func (r *InMemoryCommentRepository) Add(ctx context.Context, c *mod.CommentDTO) (int64, error) {
	r.m.Lock()
	defer r.m.Unlock()
	comments, ok := r.c[c.PostId]
	if !ok {
		return 0, errors.New("there is no such post")
	}
	c.Id = r.idGen
	r.idGen++
	ind, ok := searchIndex(comments, c.ParentId)
	if !ok {
		comments = append(comments, c)
		copy(comments[1:], comments)
		comments[0] = c

	}
	comments = append(comments[:ind+1], comments[ind:]...)
	comments[ind] = c
	r.ch <- *c
	return c.Id, nil
}

func searchIndex(data []*mod.CommentDTO, id int64) (int, bool) {
	for i, c := range data {
		if c.Id == id {
			return i, true
		}
	}
	return -1, false
}

func (r *InMemoryCommentRepository) GetAllOfPost(ctx context.Context, idPost, offset int64, limit int32) ([]*mod.CommentDTO, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	comments, ok := r.c[idPost]
	if !ok {
		return nil, errors.New("there is no such post")
	}
	ind, ok := searchIndex(comments, offset)
	if !ok {
		return nil, errors.New("there is no such parent comment")
	}
	var result []*mod.CommentDTO
	copy(comments[ind:ind+int(limit)], result)
	return result, nil
}
