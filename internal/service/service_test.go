package service_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/mi-raf/comment-project/graph/model"
	"github.com/mi-raf/comment-project/internal/models"
	"github.com/mi-raf/comment-project/internal/service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

var cc chan models.CommentDTO

type ServiceTestSuite struct {
	suite.Suite
	ctx    context.Context
	ps     *service.PostService
	pr     *MockPostRepository
	cr     *MockCommentRepository
	closer func()
}

func (suite *ServiceTestSuite) SetupSuite() {
	suite.ctx = context.Background()
}

func (suite *ServiceTestSuite) SetupTest() {
	cc = make(chan models.CommentDTO, 10)
	suite.cr = new(MockCommentRepository)
	suite.pr = new(MockPostRepository)
	suite.ps, suite.closer = service.NewPostService(suite.ctx, suite.pr, suite.cr, &service.Config{cc})
}

func (suite *ServiceTestSuite) TearDownTest() {
	suite.closer()
	close(cc)
}

// Tests

func (s *ServiceTestSuite) TestGetAllPosts() {
	// given
	var testPosts []*models.PostDTO
	for i := 0; i < 5; i++ {
		var f models.PostDTO
		gofakeit.Struct(&f)
		testPosts = append(testPosts, &f)
	}
	s.pr.On("GetAll", s.ctx, int64(0), 5).Return(testPosts, nil)

	// when
	res, err := s.ps.GetAllPosts(s.ctx, 0, 5)

	// then
	s.NoError(err)
	var expected []*model.ShortPost
	for _, r := range testPosts {
		expected = append(expected, &model.ShortPost{ID: fmti64(r.Id), Author: r.Author, Title: r.Title})
	}
	s.ElementsMatch(expected, res)
}

func (s *ServiceTestSuite) TestGetAllPostsReturnError() {
	// given
	s.pr.On("GetAll", s.ctx, int64(0), 5).Return(nil, errors.New("Test error"))

	// when
	_, err := s.ps.GetAllPosts(s.ctx, 0, 5)

	// then
	s.ErrorIs(err, service.ErrDatabase)
}

func (s *ServiceTestSuite) TestGetAllPostsReturnNil() {
	// given
	s.pr.On("GetAll", s.ctx, int64(0), 5).Return(nil, nil)

	// when
	res, err := s.ps.GetAllPosts(s.ctx, 0, 5)

	// then
	s.Nil(res)
	s.NoError(err)
}

func (s *ServiceTestSuite) TestGetPost() {
	// given
	var f models.PostDTO
	gofakeit.Struct(&f)
	s.pr.On("Get", s.ctx, int64(1)).Return(&f, nil)

	var testComments []*models.CommentDTO
	for i := 0; i < 5; i++ {
		var f models.CommentDTO
		gofakeit.Struct(&f)
		testComments = append(testComments, &f)
	}
	s.cr.On("GetAllOfPost", s.ctx, int64(1), int64(0), 5).Return(testComments, nil)

	// when
	post, err := s.ps.Post(s.ctx, 1, 5)

	// then
	s.NoError(err)
	var expectedComments []*model.CommentConnection
	for _, c := range testComments {
		expectedComments = append(expectedComments,
			&model.CommentConnection{
				ID:       fmti64(c.Id),
				ParentID: getNullableString(c.ParentId),
				PostID:   fmti64(c.PostId),
				Level:    c.Level,
				Comment: &model.Comment{
					Author: c.Author,
					Text:   c.Text,
					Time:   c.Time,
				},
			})
	}

	ec := fmti64(testComments[len(testComments)-1].Id)
	commentResult := &model.CommentsResult{
		Comments: expectedComments,
		PageInfo: &model.PageInfo{
			EndCursor: &ec,
		},
	}
	expectedPost := &model.Post{
		ID:            fmti64(f.Id),
		Author:        f.Author,
		Title:         f.Title,
		Text:          f.Text,
		Time:          f.Time,
		IsCommentable: f.IsCommentable,
		Comments:      commentResult,
	}
	s.EqualExportedValues(expectedPost, post)
}

func (s *ServiceTestSuite) TestGetPostWithError() {
	// given
	var f models.PostDTO
	gofakeit.Struct(&f)
	s.pr.On("Get", s.ctx, int64(1)).Return(nil, errors.New("Test Error"))

	s.cr.On("GetAllOfPost", s.ctx, int64(1), int64(0), 5).Return(nil, nil)

	// when
	_, err := s.ps.Post(s.ctx, 1, 5)

	// then
	s.ErrorIs(err, service.ErrDatabase)
	s.cr.AssertNotCalled(s.T(), "GetAllOfPost", s.ctx, mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestNotExistinigPost() {
	// given
	s.pr.On("Get", s.ctx, int64(1)).Return(nil, nil)

	s.cr.On("GetAllOfPost", s.ctx, int64(1), int64(0), 5).Return(nil, nil)

	// when
	_, err := s.ps.Post(s.ctx, 1, 5)

	// then
	s.NotErrorIs(err, service.ErrDatabase)
	s.cr.AssertNotCalled(s.T(), "GetAllOfPost", s.ctx, mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestCreatePost() {
	// given
	var f model.NewPost
	gofakeit.Struct(&f)
	s.pr.On("Add", s.ctx, mock.Anything).Return(1, nil)

	// when
	res, err := s.ps.CreatePost(s.ctx, f)

	expected := &model.Post{
		ID:            "1",
		Author:        f.Author,
		Title:         f.Title,
		Text:          f.Text,
		Time:          f.Time,
		IsCommentable: *f.IsCommentable,
	}
	s.NoError(err)
	s.Equal(expected, res)
}

func (s *ServiceTestSuite) TestCreatePostError() {
	// given
	var f model.NewPost
	gofakeit.Struct(&f)
	s.pr.On("Add", s.ctx, mock.Anything).Return(0, errors.New("Test error"))

	// when
	_, err := s.ps.CreatePost(s.ctx, f)

	// then
	s.ErrorIs(err, service.ErrDatabase)
}

func (s *ServiceTestSuite) TestSubsribeAndListen() {
	// given
	var comment models.CommentDTO
	gofakeit.Struct(&comment)
	ch := s.ps.CommentSubscribe(comment.PostId)

	// when
	cc <- comment

	// then
	select {
	case actual := <-ch:
		expected := &model.CommentConnection{
			ID:       fmti64(comment.Id),
			ParentID: getNullableString(comment.ParentId),
			PostID:   fmti64(comment.PostId),
			Level:    comment.Level,
			Comment: &model.Comment{
				Author: comment.Author,
				Text:   comment.Text,
				Time:   comment.Time,
			},
		}
		s.Equal(expected, actual)
	case <-time.After(2 * time.Second):
		s.FailNow("it's been too long")
	}
}

func (s *ServiceTestSuite) TestSubsribeToExistingAndListen() {
	// given
	var comment models.CommentDTO
	gofakeit.Struct(&comment)
	ch1 := s.ps.CommentSubscribe(comment.PostId)
	ch2 := s.ps.CommentSubscribe(comment.PostId)

	// when
	cc <- comment

	// then
	s.Equal(ch1, ch2)

	select {
	case actual := <-ch1:
		expected := &model.CommentConnection{
			ID:       fmti64(comment.Id),
			ParentID: getNullableString(comment.ParentId),
			PostID:   fmti64(comment.PostId),
			Level:    comment.Level,
			Comment: &model.Comment{
				Author: comment.Author,
				Text:   comment.Text,
				Time:   comment.Time,
			},
		}
		s.Equal(expected, actual)
	case <-time.After(2 * time.Second):
		s.FailNow("it's been too long")
	}
}

func (s *ServiceTestSuite) TestGetComments() {
	// given
	var testComments []*models.CommentDTO
	for i := 0; i < 5; i++ {
		var f models.CommentDTO
		gofakeit.Struct(&f)
		testComments = append(testComments, &f)
	}
	s.cr.On("GetAllOfPost", s.ctx, int64(1), int64(0), 5).Return(testComments, nil)

	// when
	res, err := s.ps.Comments(s.ctx, 1, 5, 0)
	s.NoError(err)

	var expectedComments []*model.CommentConnection
	for _, c := range testComments {
		expectedComments = append(expectedComments,
			&model.CommentConnection{
				ID:       fmti64(c.Id),
				ParentID: getNullableString(c.ParentId),
				PostID:   fmti64(c.PostId),
				Level:    c.Level,
				Comment: &model.Comment{
					Author: c.Author,
					Text:   c.Text,
					Time:   c.Time,
				},
			})
	}

	ec := fmti64(testComments[len(testComments)-1].Id)
	expected := &model.CommentsResult{
		Comments: expectedComments,
		PageInfo: &model.PageInfo{
			EndCursor: &ec,
		},
	}
	s.EqualExportedValues(expected, res)
}

func (s *ServiceTestSuite) TestGetCommentsWithIncorrectLimit() {
	// given
	var testComments []*models.CommentDTO
	s.cr.On("GetAllOfPost", s.ctx, int64(1), int64(0), 100).Return(testComments, nil)

	// when
	_, err := s.ps.Comments(s.ctx, 1, -1, 0)
	s.NoError(err)
	_, err = s.ps.Comments(s.ctx, 1, 101, 0)
	s.NoError(err)

	// then
	s.cr.AssertNumberOfCalls(s.T(), "GetAllOfPost", 2)
}

func (s *ServiceTestSuite) TestGetCommentsWithIncorrectOffset() {
	// given
	var testComments []models.CommentDTO
	s.cr.On("GetAllOfPost", s.ctx, mock.Anything, mock.Anything, mock.Anything).Return(testComments, nil)

	// when
	_, err := s.ps.Comments(s.ctx, 1, 5, -1)

	// then
	s.Error(err)
	s.cr.AssertNotCalled(s.T(), "GetAllOfPost")
}

func (s *ServiceTestSuite) TestGetCommentsEmptyResult() {
	// given
	var testComments []*models.CommentDTO
	s.cr.On("GetAllOfPost", s.ctx, mock.Anything, mock.Anything, mock.Anything).Return(testComments, nil)

	// when
	res, err := s.ps.Comments(s.ctx, 1, 5, 0)

	// then
	s.NoError(err)
	s.Len(res.Comments, 0)
	s.Equal(&model.PageInfo{EndCursor: nil}, res.PageInfo)
}

func (s *ServiceTestSuite) TestCreateComment() {
	// given
	var f model.NewComment
	gofakeit.Struct(&f)
	s.pr.On("IsCommentable", s.ctx, mock.Anything).Return(true, nil)
	s.cr.On("Add", s.ctx, mock.Anything).Return(1, nil)
	// when
	res, err := s.ps.CreateComment(s.ctx, 1, 0, f)

	// then
	s.NoError(err)
	s.Equal(int64(1), res)
	s.pr.AssertExpectations(s.T())
	s.cr.AssertExpectations(s.T())
}

func (s *ServiceTestSuite) TestCreateCommentNotCommentable() {
	// given
	var f model.NewComment
	gofakeit.Struct(&f)
	s.pr.On("IsCommentable", s.ctx, mock.Anything).Return(false, nil)
	s.cr.On("Add", s.ctx, mock.Anything).Return(1, nil)
	// when
	_, err := s.ps.CreateComment(s.ctx, 1, 0, f)

	// then
	s.Error(err)
	s.pr.AssertExpectations(s.T())
	s.cr.AssertNotCalled(s.T(), "Add", s.ctx, mock.Anything)
}

func (s *ServiceTestSuite) TestCreateCommentListen() {
	// given
	var postId int64 = 1
	ch := s.ps.CommentSubscribe(postId)

	var f model.NewComment
	gofakeit.Struct(&f)
	s.pr.On("IsCommentable", s.ctx, mock.Anything).Return(true, nil)
	s.cr.On("Add", s.ctx, mock.Anything).Return(1, nil)
	// when
	commentId, err := s.ps.CreateComment(s.ctx, postId, 0, f)

	// then
	s.NoError(err)
	select {
	case actual := <-ch:
		expected := &model.CommentConnection{
			ID:       fmti64(commentId),
			ParentID: nil,
			PostID:   fmti64(postId),
			Level:    0,
			Comment: &model.Comment{
				Author: f.Author,
				Text:   f.Text,
				Time:   f.Time,
			},
		}
		s.Equal(expected, actual)
	case <-time.After(2 * time.Second):
		s.FailNow("it's been too long")
	}
}

func fmti64(data int64) string {
	return strconv.FormatInt(data, 10)
}

// Mocks

type MockPostRepository struct {
	mock.Mock
}

type MockCommentRepository struct {
	mock.Mock
}

func (p *MockPostRepository) Add(ctx context.Context, post *models.PostDTO) (int64, error) {
	args := p.Called(ctx, post)
	return int64(args.Int(0)), args.Error(1)
}

func (p *MockPostRepository) GetAll(ctx context.Context, offset int64, limit int) ([]*models.PostDTO, error) {
	args := p.Called(ctx, offset, limit)
	f := args.Get(0)
	if f == nil {
		return nil, args.Error(1)
	}
	return f.([]*models.PostDTO), args.Error(1)
}
func (p *MockPostRepository) Get(ctx context.Context, id int64) (*models.PostDTO, error) {
	args := p.Called(ctx, id)
	f := args.Get(0)
	if f == nil {
		return nil, args.Error(1)
	}
	return f.(*models.PostDTO), args.Error(1)
}
func (p *MockPostRepository) IsCommentable(ctx context.Context, id int64) (bool, error) {
	args := p.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (p *MockCommentRepository) Add(ctx context.Context, c *models.CommentDTO) (int64, error) {
	args := p.Called(ctx, c)
	cc <- models.CommentDTO{
		Id:       int64(args.Int(0)),
		PostId:   c.PostId,
		ParentId: c.ParentId,
		Author:   c.Author,
		Level:    0,
		Text:     c.Text,
		Time:     c.Time,
	}
	return int64(args.Int(0)), args.Error(1)
}

func (p *MockCommentRepository) GetAllOfPost(ctx context.Context, idPost int64, offset int64, limit int) ([]*models.CommentDTO, error) {
	args := p.Called(ctx, idPost, offset, limit)
	return args.Get(0).([]*models.CommentDTO), args.Error(1)
}
func (p *MockCommentRepository) Get(ctx context.Context, id int64) (models.CommentDTO, error) {
	args := p.Called(ctx, id)
	return args.Get(0).(models.CommentDTO), args.Error(1)
}

func getNullableString(data int64) *string {
	if data == 0 {
		return nil
	}
	res := strconv.FormatInt(data, 10)
	return &res
}

func TestCustomerRepoTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}
