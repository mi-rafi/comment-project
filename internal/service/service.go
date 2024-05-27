package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/mi-raf/comment-project/graph/model"
	"github.com/mi-raf/comment-project/internal/database"
	"github.com/mi-raf/comment-project/internal/models"
	"github.com/rs/zerolog/log"
)

const (
	MAX_LIMIT    = 100
	CHANELS_SIZE = 10
)

var ErrDatabase = errors.New("errors while loading or saving data")
var ErrClientData = errors.New("incorrect client data")

type (
	PostService struct {
		lock sync.RWMutex
		pc   map[int64]chan *model.CommentConnection
		ch   <-chan models.CommentDTO
		p    database.PostRepository
		c    database.CommentRepository
	}

	Config struct {
		CommentChan <-chan models.CommentDTO
	}
)

func NewCommentChan() (chan models.CommentDTO, func()) {
	ch := make(chan models.CommentDTO, CHANELS_SIZE)
	return ch, func() {
		close(ch)
	}
}

func NewPostService(ctx context.Context, p database.PostRepository, c database.CommentRepository, cfg *Config) (*PostService, func()) {
	ps := &PostService{pc: make(map[int64]chan *model.CommentConnection), ch: cfg.CommentChan, p: p, c: c}
	go ps.ListenComments(ctx)
	return ps, func() {
		for _, ch := range ps.pc {
			close(ch)
		}
	}
}

func (ps *PostService) GetAllPosts(ctx context.Context, offset int64, limit int) ([]*model.ShortPost, error) {
	log.Debug().Int64("offset", offset).Int("limit", limit).Msg("get all posts")
	if offset < 0 {
		log.Error().Int64("after", offset).Msg("incorrect after")
		return nil, fmt.Errorf("after %d should be more than 0", offset)
	}
	posts, err := ps.p.GetAll(ctx, offset, getLimit(limit))
	if err != nil {
		log.Error().Err(err).Msg("error while loading all posts")
		return nil, ErrDatabase
	}
	log.Debug().Int("result size", len(posts)).Msg("posts returned")
	var res []*model.ShortPost
	for _, p := range posts {
		res = append(res, &model.ShortPost{
			Author: p.Author,
			ID:     fmti64(p.Id),
			Title:  p.Title,
		})
	}
	return res, nil
}

func (ps *PostService) CommentSubscribe(postId int64) <-chan *model.CommentConnection {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	if ch, ok := ps.pc[postId]; ok {
		log.Debug().Int64("post id", postId).Msg("return existing")
		return ch
	}
	ch := make(chan *model.CommentConnection, CHANELS_SIZE)
	ps.pc[postId] = ch
	log.Debug().Int64("post id", postId).Msg("return created")
	return ch
}

func (ps *PostService) ListenComments(ctx context.Context) {
	for {
		select {
		case c, ok := <-ps.ch:
			if !ok {
				return
			}
			log.Debug().Int64("id", c.Id).Msg("new comment recevied")
			pId := c.PostId
			ps.lock.RLock()
			if ch, ok := ps.pc[pId]; ok {
				ch <- &model.CommentConnection{
					Comment:  &model.Comment{Author: c.Author, Text: c.Text, Time: c.Time},
					ID:       fmti64(c.Id),
					ParentID: getNullableString(c.ParentId),
					Level:    c.Level,
					PostID:   fmti64(c.PostId),
				}
				log.Debug().Int64("post id", c.Id).Msg("comment sent to channel")
			}
			ps.lock.RUnlock()
		case <-ctx.Done():
			log.Info().Msg("context closed")
			return
		}
	}
}

func (ps *PostService) CreatePost(ctx context.Context, input model.NewPost) (*model.Post, error) {
	ic := true
	if input.IsCommentable != nil {
		ic = *input.IsCommentable
	}
	p := &models.PostDTO{
		Author:        input.Author,
		Text:          input.Text,
		Title:         input.Title,
		IsCommentable: ic,
		Time:          input.Time,
	}
	id, err := ps.p.Add(ctx, p)
	if err != nil {
		log.Error().Err(err).Str("title", p.Title).Msg("error while creating post")
		return nil, ErrDatabase
	}
	log.Debug().Int64("id", id).Msg("post created")
	return &model.Post{
		ID:            fmti64(id),
		Author:        input.Author,
		Text:          input.Text,
		Title:         input.Title,
		IsCommentable: ic,
		Time:          input.Time,
	}, nil
}

func (ps *PostService) CreateComment(ctx context.Context, postId int64, pc int64, comment model.NewComment) (int64, error) {
	log.Debug().Int64("post id", postId).Int64("parent comment id", pc).Msg("creating new comment")
	ic, err := ps.p.IsCommentable(ctx, postId)
	if err != nil {
		log.Error().Err(err).Int64("post id", postId).Msg("error while getting info about commentable")
		return 0, ErrDatabase
	}
	if !ic {
		log.Debug().Int64("post id", postId).Msg("post is not commentable")
		return 0, fmt.Errorf("can not comment post: %d", postId)
	}
	id, err := ps.c.Add(ctx, models.CommentDTO{PostId: postId, ParentId: pc, Author: comment.Author, Text: comment.Text, Time: comment.Time})
	if err != nil {
		log.Error().Err(err).Int64("post id", postId).Str("author", comment.Author).Str("text", comment.Text).Msg("error while saving comment info")
		return 0, ErrDatabase
	}
	log.Debug().Int64("post id", postId).Int64("comment id", id).Msg("comment created")
	return id, nil
}

func (ps *PostService) Post(ctx context.Context, id int64, limit int) (*model.Post, error) {
	log.Debug().Int64("post id", id).Msg("getting info about post")
	post, err := ps.p.Get(ctx, id)
	if err != nil {
		log.Error().Err(err).Int64("post id", id).Msg("error while getting info about post")
		return nil, ErrDatabase
	}
	if post == nil {
		return nil, nil
	}
	limit = getLimit(limit)
	log.Debug().Int64("post id", id).Int("limit", limit).Msg("getting info about comments of post")
	cs, err := ps.c.GetAllOfPost(ctx, id, 0, limit)
	if err != nil {
		log.Error().Err(err).Int64("post id", id).Msg("error while getting comments for post in getting post")
		return nil, ErrDatabase
	}
	cr := mapComments(cs)
	log.Debug().Int64("post id", id).Int("result size", len(cs)).Msg("result comments of post info")
	return &model.Post{ID: strconv.FormatInt(post.Id, 10), Author: post.Author, Title: post.Title, Text: post.Text, Time: post.Time, IsCommentable: post.IsCommentable, Comments: cr}, nil

}

func (ps *PostService) Comments(ctx context.Context, postID int64, limit int, offset int64) (*model.CommentsResult, error) {
	log.Debug().Int64("post id", postID).Int("limit", limit).Int64("offset", offset).Msg("getting info about comments of post with offset")
	if offset < 0 {
		log.Error().Int64("after", offset).Msg("incorrect after")
		return nil, fmt.Errorf("after %d should be more than 0", offset)
	}
	cs, err := ps.c.GetAllOfPost(ctx, postID, offset, getLimit(limit))
	if err != nil {
		log.Error().Err(err).Int64("post id", postID).Msg("error while getting comments")
		return nil, ErrDatabase
	}
	log.Debug().Int64("post id", postID).Int("result size", len(cs)).Msg("result comments info")
	return mapComments(cs), nil
}

func fmti64 (data int64) string {
	return strconv.FormatInt(data, 10)
}

func mapComments(c []*models.CommentDTO) *model.CommentsResult {
	if c == nil || len(c) == 0 {
		return &model.CommentsResult{
			Comments: nil,
			PageInfo: &model.PageInfo{
				EndCursor: nil,
			},
		}
	}
	var res []*model.CommentConnection
	for _, cm := range c {
		last := &model.CommentConnection{
			ID:       fmti64(cm.Id),
			PostID:   fmti64(cm.PostId),
			ParentID: getNullableString(cm.ParentId),
			Level:    cm.Level,
			Comment:  &model.Comment{Author: cm.Author, Text: cm.Text},
		}
		res = append(res, last)
	}
	pgi :=  &model.PageInfo{}
	if len(res) > 0 {
		pgi.EndCursor = &res[len(res)-1].ID
	}
	return &model.CommentsResult{Comments: res, PageInfo: pgi}
}

func getNullableString(data int64) *string {
	if data == 0 {
		return nil
	}
	res := strconv.FormatInt(data, 10)
	return &res
}

func getLimit(l int) int {
	if l < 1 || l > MAX_LIMIT {
		log.Debug().Int("user limit", l).Msg("switch to default limit")
		return MAX_LIMIT
	}
	return l
}
