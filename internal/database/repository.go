package database

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	mod "github.com/mi-raf/comment-project/internal/models"
)

const (
	insertPost    = "INSERT INTO Post (author, title, text_p, comm) VALUES ($1, $2, $3, $4)"
	insertComment = "INSERT INTO Comment (id_p, parent, author, text_c) VALUES ($1, $2, $3, $4)"
	searchAllPost = `SELECT id_p, author, title, text_p, comm 
	FROM Post 
	ORDER BY title
	LIMIT $1
	OFFSET $2`
	searchPost = `SELECT  author, title, text_p, comm 
	FROM Post 
	WHERE id_p = $1`
)

const (
	STARTCAP = 128
)

type (
	PostRepository interface {
		//Delete(ctx context.Context) error
		Add(ctx context.Context, post mod.PostDTO) error
		GetAll(ctx context.Context, offset, limit int) ([]mod.PostDTO, error)
		Get(ctx context.Context, id int, limit int) (*mod.PostDTO, error)
	}

	CommentRepository interface {
		Add(ctx context.Context, c mod.CommentDTO) error
		GetAllOfPost(ctx context.Context, idPost int32, offset, limit int) ([]mod.CommentDTO, error)
		Get(ctx context.Context, id int32) (mod.CommentDTO, error)
	}

	PgPostRepository struct {
		pool *pgxpool.Pool
	}

	PgCommentRepository struct {
		pool *pgxpool.Pool
	}

	InMemoryPostRepository struct {
		posts []mod.PostDTO
		m     sync.RWMutex
	}

	InMemoryCommentRepository struct {
		c map[int64]mod.CommentDTO
	}

	PostRepositoryProvider struct {
	}

	CommentRepositoryProvider struct {
	}
)

func NewPostRepositoryProvider(ctx context.Context, p *pgxpool.Pool, inMemory bool) (PostRepository, error) {
	if inMemory {
		return NewInMemnoryPostRepository(), nil
	}
	r, err := NewPgCommentRepository(ctx, p)
	return r, err
}

func NewPgPostRepository(ctx context.Context, p *pgxpool.Pool) (*PgPostRepository, error) {
	return &PgPostRepository{pool: p}, nil
}

func (r *PgPostRepository) Close(ctx context.Context) error {
	r.pool.Close()
	return nil
}

func (r *PgPostRepository) Add(ctx context.Context, post mod.PostDTO) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("can't open transaction for add post")
		return err
	}

	defer func() {
		err = tx.Rollback(ctx)
		if !errors.Is(err, pgx.ErrTxClosed) {
			log.Error().Err(err).Msg("Undefinded error in tx")
		}
	}()

	log.Debug().Interface("post", post).Msg("add post")
	_, err = tx.Exec(ctx, insertPost, post.Author, post.Title, post.Text, post.IsComments)

	if err != nil {
		log.Error().Err(err).Msg("error insert received")
		return err
	}
	return tx.Commit(ctx)
}

func (r *PgPostRepository) GetAll(ctx context.Context, offset, limit int) ([]mod.PostDTO, error) {
	rows, err := r.pool.Query(ctx, searchAllPost, limit, offset)
	if err != nil {
		return nil, err
	}

	posts := make([]mod.PostDTO, 0)

	for rows.Next() {
		var p mod.PostDTO
		err = rows.Scan(&p.Id, &p.Author, &p.Title, &p.Text, &p.IsComments)
		if err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}

	return posts, nil
}

func (r *PgPostRepository) Get(ctx context.Context, id int32, offset, limit int) (*mod.PostDTO, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("can't open transaction for add")
		return nil, nil, err
	}

	defer func() {
		err = tx.Rollback(ctx)
		if !errors.Is(err, pgx.ErrTxClosed) {
			log.Error().Err(err).Msg("Undefinded error in tx")
		}
	}()

	row := tx.QueryRow(ctx, searchPost, id)
	p := mod.PostDTO{}
	err = row.Scan(&p.Author, &p.Title, &p.Text, &p.IsComments)
	if err == sql.ErrNoRows {
		log.Error().Msg("no post with this id")
	}

	repCom, err := NewPgCommentRepository(ctx, r.pool)

	c, err := repCom.GetAllOfPost(ctx, id, offset, limit)

	return &p, c, err
}

func NewPgCommentRepository(ctx context.Context, p *pgxpool.Pool) (*PgCommentRepository, error) {
	return &PgCommentRepository{pool: p}, nil
}

func (r *PgCommentRepository) Add(ctx context.Context, c mod.CommentDTO) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("can't open transaction for add comment")
		return err
	}

	defer func() {
		err = tx.Rollback(ctx)
		if !errors.Is(err, pgx.ErrTxClosed) {
			log.Error().Err(err).Msg("Undefinded error in tx")
		}
	}()

	log.Debug().Interface("comment", c).Msg("add comment")
	_, err = tx.Exec(ctx, insertComment, c.PostId, c.ParentId, c.Author, c.Text)

	if err != nil {
		log.Error().Err(err).Msg("error insert received")
		return err
	}
	return tx.Commit(ctx)
}

func (r *PgCommentRepository) Get(ctx context.Context, id int32) (mod.CommentDTO, error) {
	panic("jhds")
}

func (r *PgCommentRepository) GetAllOfPost(ctx context.Context, idPost int32, offset, limit int) ([]mod.CommentDTO, error) {
	panic("fff")
}

func NewInMemnoryPostRepository() (*InMemoryPostRepository, error) {
	p := make([]mod.PostDTO, 0, STARTCAP)
	return &InMemoryPostRepository{posts: p}, nil
}

func (r *InMemoryPostRepository) Add(ctx context.Context, post mod.PostDTO) error {
	r.m.Lock()
	defer r.m.Unlock()
	post.Id = int32(len(r.posts))
	r.posts = append(r.posts, post)
	return nil
}

func (r *InMemoryPostRepository) GetAll(ctx context.Context, offset, limit int) ([]mod.PostDTO, error)
func (r *InMemoryPostRepository) Get(ctx context.Context, id int, limit int) (*mod.PostDTO, []mod.CommentDTO, error)
