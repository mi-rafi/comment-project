package database_test

import (
	"context"
	"math/rand/v2"
	"strconv"
	"time"

	"sort"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/mi-raf/comment-project/graph/model"
	"github.com/mi-raf/comment-project/internal/database"
	"github.com/mi-raf/comment-project/internal/models"
	"github.com/rs/zerolog/log"

	"github.com/stretchr/testify/suite"
)

type PgPostRepositoryMemoryTestSuite struct {
	suite.Suite
	r   database.PostRepository
	ctx context.Context
}

func (suite *PgPostRepositoryMemoryTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	var err error
	suite.r = database.NewInMemnoryPostRepository()
	suite.NoError(err)
}

func (s *PgPostRepositoryMemoryTestSuite) TestCreatePost() {
	// given
	var f models.PostDTO
	gofakeit.Struct(&f)

	// when
	id, err := s.r.Add(s.ctx, &f)

	// then
	s.NoError(err)
	s.Equal(id, int64(1))
}

func (s *PgPostRepositoryMemoryTestSuite) TestGetPost() {
	// given
	var f models.PostDTO
	gofakeit.Struct(&f)
	id, err := s.r.Add(s.ctx, &f)
	f.Id = id
	s.NoError(err)

	// when
	actual, err := s.r.Get(s.ctx, id)

	// then
	s.NoError(err)
	s.Equal(f, *actual)
}

func (s *PgPostRepositoryMemoryTestSuite) TestGetPostEmpty() {

	// when
	actual, err := s.r.Get(s.ctx, -1)

	// then
	s.NoError(err)
	s.Nil(actual)
}

func (s *PgPostRepositoryMemoryTestSuite) TestGetSmallPosts() {
	// given
	var expected []*models.PostDTO
	for i := 0; i < 10; i++ {
		var f models.PostDTO
		gofakeit.Struct(&f)
		id, err := s.r.Add(s.ctx, &f)
		s.NoError(err)
		f.Id = id
		expected = append(expected, &f)
	}

	// when
	actual, err := s.r.GetAll(s.ctx, 0, 30)

	log.Debug().Int("actual len", len(actual)).Msgf("actual posts: %+v", actual)
	// then
	s.NoError(err)
	// s.ElementsMatch(expected, actual)
}

func (s *PgPostRepositoryMemoryTestSuite) TestGetSmallPostsPagination() {
	// given
	var expected []*model.ShortPost
	for i := 0; i < 10; i++ {
		var f models.PostDTO
		gofakeit.Struct(&f)
		id, err := s.r.Add(s.ctx, &f)
		s.NoError(err)
		expected = append(expected, &model.ShortPost{
			ID:     strconv.FormatInt(id, 10),
			Author: f.Author,
			Title:  f.Title,
		})
	}

	// when
	actual, err := s.r.GetAll(s.ctx, 5, 10)

	// then
	log.Debug().Int("actual len", len(actual)).Msgf(" pagination actual posts: %+v", actual)
	s.NoError(err)
	// s.ElementsMatch(expected[5:], actual)
}

func TestPgPostRepositoryMemoryTestSuite(t *testing.T) {
	suite.Run(t, new(PgPostRepositoryMemoryTestSuite))
}

type PgCommentRepositoryMemoryTestSuite struct {
	suite.Suite
	r   database.CommentRepository
	ctx context.Context
	ch  chan models.CommentDTO
}

func (suite *PgCommentRepositoryMemoryTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	var err error
	c := make(chan<- models.CommentDTO, 10)
	suite.r = database.NewInMemnoryCommentRepository(c)
	suite.NoError(err)
}

func (s *PgCommentRepositoryMemoryTestSuite) TearDownSuite() {
	close(s.ch)
}

func (s *PgCommentRepositoryMemoryTestSuite) TestCreateComment() {
	// given
	var f models.CommentDTO
	gofakeit.Struct(&f)
	f.ParentId = 0
	f.PostId = int64(rand.IntN(10))

	// when
	log.Debug().Int64("post id", f.PostId).Msg("random post")
	_, err := s.r.Add(s.ctx, &f)

	// then
	s.NoError(err)
}
func (s *PgCommentRepositoryMemoryTestSuite) TestGetComments() {
	// given
	var expected []*models.CommentDTO
	postID := int64(rand.IntN(10))
	now := time.Now()
	for i := 0; i < 10; i++ {
		var f models.CommentDTO
		gofakeit.Struct(&f)
		f.PostId = postID
		f.ParentId = 0
		f.Time = now
		id, err := s.r.Add(s.ctx, &f)
		f.Level = 1
		f.Id = id
		s.NoError(err)
		expected = append(expected, &f)
	}

	// when
	actual, err := s.r.GetAllOfPost(s.ctx, postID, 0, 10)

	// then
	s.NoError(err)
	sort.Slice(actual, func(i, j int) bool {
		return actual[i].Id < actual[j].Id
	})
	for _, d := range actual {
		d.Time = now
	}

	s.ElementsMatch(expected, actual)
}

func (s *PgCommentRepositoryMemoryTestSuite) TestGetEmptyComments() {

	// when
	actual, err := s.r.GetAllOfPost(s.ctx, 1, 0, 10)

	// then
	s.NoError(err)
	s.Len(actual, 0)
}

func TestPgCommentRepositoryMemoryTestSuite(t *testing.T) {
	suite.Run(t, new(PgCommentRepositoryMemoryTestSuite))
}
