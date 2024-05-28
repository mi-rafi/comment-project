package database_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	//"github.com/go-delve/delve/pkg/dwarf/regnum"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mi-raf/comment-project/internal/database"
	"github.com/mi-raf/comment-project/internal/models"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PgPostRepositoryTestSuite struct {
	suite.Suite
	r           database.PostRepository
	pgContainer *postgres.PostgresContainer
	ctx         context.Context
}

func (suite *PgPostRepositoryTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	var err error
	suite.pgContainer, err = postgres.RunContainer(suite.ctx,
		testcontainers.WithImage("postgres:15.3-alpine"),
		postgres.WithInitScripts(filepath.Join("..", "..", "testdata", "init-db.sql")),
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	suite.NoError(err)
	connStr, err := suite.pgContainer.ConnectionString(suite.ctx, "sslmode=disable")

	suite.NoError(err)
	p, err := pgxpool.New(suite.ctx, connStr)
	suite.NoError(err)
	suite.r = database.NewPgPostRepository(suite.ctx, p)
	suite.NoError(err)
	err = suite.pgContainer.CopyFileToContainer(suite.ctx, filepath.Join("..", "..", "testdata", "drop-info.sql"), "/drop-info.sql", int64(os.ModePerm.Perm()))
	suite.NoError(err)
}

// func (suite *PgPostRepositoryTestSuite) SetupTest() {
// 	_, _, err := suite.pgContainer.Exec(suite.ctx, []string{"psql", "-U", "postgres", "-d", "test-db", "-f", "/insert-cars.sql"})
// 	suite.NoError(err)
// }

func (suite *PgPostRepositoryTestSuite) TearDownTest() {
	_, _, err := suite.pgContainer.Exec(suite.ctx, []string{"psql", "-U", "postgres", "-d", "test-db", "-f", "/drop-info.sql"})
	suite.NoError(err)
}

func (s *PgPostRepositoryTestSuite) TearDownSuite() {
	err := s.pgContainer.Terminate(s.ctx)
	s.NoError(err)
}

func (s *PgPostRepositoryTestSuite) TestCreatePost() {
	// given
	var f models.PostDTO
	gofakeit.Struct(&f)

	// when
	_, err := s.r.Add(s.ctx, &f)

	// then
	s.NoError(err)
}

func (s *PgPostRepositoryTestSuite) TestGetPost() {
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

func (s *PgPostRepositoryTestSuite) TestGetPostEmpty() {

	// when
	actual, err := s.r.Get(s.ctx, -1)

	// then
	s.NoError(err)
	s.Nil(actual)
}

func (s *PgPostRepositoryTestSuite) TestGetSmallPostsPagination() {
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
	actual, err := s.r.GetAll(s.ctx, 5, 10)

	// then
	s.NoError(err)
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].Time.After(expected[j].Time)
	})
	s.ElementsMatch(expected[5:], actual)
}

func TestPgPostRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(PgPostRepositoryTestSuite))
}

type PgCommentRepositoryTestSuite struct {
	suite.Suite
	r           database.CommentRepository
	pgContainer *postgres.PostgresContainer
	pool        *pgxpool.Pool
	ctx         context.Context
	ch          chan models.CommentDTO
}

func (suite *PgCommentRepositoryTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	var err error
	suite.pgContainer, err = postgres.RunContainer(suite.ctx,
		testcontainers.WithImage("postgres:15.3-alpine"),
		postgres.WithInitScripts(filepath.Join("..", "..", "testdata", "init-db.sql")),
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	suite.NoError(err)
	connStr, err := suite.pgContainer.ConnectionString(suite.ctx, "sslmode=disable")

	suite.NoError(err)
	p, err := pgxpool.New(suite.ctx, connStr)
	suite.NoError(err)
	suite.pool = p
	suite.ch = make(chan models.CommentDTO, 10)
	suite.r = database.NewPgCommentRepository(suite.ctx, p, suite.ch)
	suite.NoError(err)

	err = suite.pgContainer.CopyFileToContainer(suite.ctx, filepath.Join("..", "..", "testdata", "add-posts.sql"), "/add-posts.sql", int64(os.ModePerm.Perm()))
	suite.NoError(err)
	err = suite.pgContainer.CopyFileToContainer(suite.ctx, filepath.Join("..", "..", "testdata", "drop-info.sql"), "/drop-info.sql", int64(os.ModePerm.Perm()))
	suite.NoError(err)
}

func (suite *PgCommentRepositoryTestSuite) SetupTest() {
	_, _, err := suite.pgContainer.Exec(suite.ctx, []string{"psql", "-U", "postgres", "-d", "test-db", "-f", "/add-posts.sql"})
	suite.NoError(err)
}

func (suite *PgCommentRepositoryTestSuite) TearDownTest() {
	_, _, err := suite.pgContainer.Exec(suite.ctx, []string{"psql", "-U", "postgres", "-d", "test-db", "-f", "/drop-info.sql"})
	suite.NoError(err)
}

func (s *PgCommentRepositoryTestSuite) TearDownSuite() {
	err := s.pgContainer.Terminate(s.ctx)
	s.NoError(err)
	close(s.ch)
}

func (s *PgCommentRepositoryTestSuite) TestCreateComment() {
	// given
	var f models.CommentDTO
	gofakeit.Struct(&f)
	f.ParentId = 0
	f.PostId = s.getRandomPostId()

	// when
	_, err := s.r.Add(s.ctx, &f)

	// then
	s.NoError(err)
}

func (s *PgCommentRepositoryTestSuite) TestCreateCommentAndListen() {
	// given
	var f models.CommentDTO
	gofakeit.Struct(&f)
	f.ParentId = 0
	f.PostId = s.getRandomPostId()

	// when
	_, err := s.r.Add(s.ctx, &f)

	// then
	s.NoError(err)
	select {
	case actual := <-s.ch:

		s.Equal(f, actual)
	case <-time.After(2 * time.Second):
		s.FailNow("it's been too long")
	}
}

func (s *PgCommentRepositoryTestSuite) TestGetComments() {
	// given
	var expected []*models.CommentDTO
	postID := s.getRandomPostId()
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

func (s *PgCommentRepositoryTestSuite) TestGetCommentsOrdered() {
	// given
	var expected []*models.CommentDTO
	postID := int64(1)
	var r1 models.CommentDTO
	gofakeit.Struct(&r1)
	r1.PostId = postID
	r1.ParentId = 0
	id, err := s.r.Add(s.ctx, &r1)
	r1.Id = id
	s.NoError(err)
	expected = append(expected, &r1)

	var cr1 models.CommentDTO
	gofakeit.Struct(&cr1)
	cr1.PostId = postID
	cr1.ParentId = r1.Id
	id, err = s.r.Add(s.ctx, &cr1)
	cr1.Id = id
	s.NoError(err)
	expected = append(expected, &cr1)

	var cr2 models.CommentDTO
	gofakeit.Struct(&cr2)
	cr2.PostId = postID
	cr2.ParentId = r1.Id
	id, err = s.r.Add(s.ctx, &cr2)
	cr2.Id = id
	s.NoError(err)
	expected = append(expected, &cr2)

	var cr3 models.CommentDTO
	gofakeit.Struct(&cr3)
	cr3.PostId = postID
	cr3.ParentId = r1.Id
	id, err = s.r.Add(s.ctx, &cr3)
	cr3.Id = id
	s.NoError(err)
	expected = append(expected, &cr3)

	var cr11 models.CommentDTO
	gofakeit.Struct(&cr11)
	cr11.PostId = postID
	cr11.ParentId = cr1.Id
	id, err = s.r.Add(s.ctx, &cr11)
	cr11.Id = id
	s.NoError(err)
	expected = append(expected, &cr11)

	var cr12 models.CommentDTO
	gofakeit.Struct(&cr12)
	cr12.PostId = postID
	cr12.ParentId = cr1.Id
	id, err = s.r.Add(s.ctx, &cr12)
	cr12.Id = id
	s.NoError(err)
	expected = append(expected, &cr12)

	var cr21 models.CommentDTO
	gofakeit.Struct(&cr21)
	cr21.PostId = postID
	cr21.ParentId = cr2.Id
	id, err = s.r.Add(s.ctx, &cr21)
	cr21.Id = id
	s.NoError(err)
	expected = append(expected, &cr21)

	var r2 models.CommentDTO
	gofakeit.Struct(&r2)
	r2.PostId = postID
	r2.ParentId = 0
	id, err = s.r.Add(s.ctx, &r2)
	r2.Id = id
	s.NoError(err)
	expected = append(expected, &r2)

	var crr1 models.CommentDTO
	gofakeit.Struct(&crr1)
	crr1.PostId = postID
	crr1.ParentId = r2.Id
	id, err = s.r.Add(s.ctx, &crr1)
	crr1.Id = id
	s.NoError(err)
	expected = append(expected, &crr1)

	var crr2 models.CommentDTO
	gofakeit.Struct(&crr2)
	crr2.PostId = postID
	crr2.ParentId = r2.Id
	id, err = s.r.Add(s.ctx, &crr2)
	crr2.Id = id
	s.NoError(err)
	expected = append(expected, &crr2)

	var crr12 models.CommentDTO
	gofakeit.Struct(&crr12)
	crr12.PostId = postID
	crr12.ParentId = crr1.Id
	id, err = s.r.Add(s.ctx, &crr12)
	crr12.Id = id
	s.NoError(err)
	expected = append(expected, &crr12)

	// when
	actual, err := s.r.GetAllOfPost(s.ctx, postID, 0, 5)

	// then
	s.NoError(err)
	s.ElementsMatch(expected[0:5], actual)
}

func (s *PgCommentRepositoryTestSuite) TestGetCommentsOrderedPagination() {
	// given
	var expected []*models.CommentDTO
	postID := int64(1)
	var r1 models.CommentDTO
	gofakeit.Struct(&r1)
	r1.PostId = postID
	r1.ParentId = 0
	id, err := s.r.Add(s.ctx, &r1)
	r1.Id = id
	s.NoError(err)
	expected = append(expected, &r1)

	var cr1 models.CommentDTO
	gofakeit.Struct(&cr1)
	cr1.PostId = postID
	cr1.ParentId = r1.Id
	id, err = s.r.Add(s.ctx, &cr1)
	cr1.Id = id
	s.NoError(err)
	expected = append(expected, &cr1)

	var cr2 models.CommentDTO
	gofakeit.Struct(&cr2)
	cr2.PostId = postID
	cr2.ParentId = r1.Id
	id, err = s.r.Add(s.ctx, &cr2)
	cr2.Id = id
	s.NoError(err)
	expected = append(expected, &cr2)

	var cr3 models.CommentDTO
	gofakeit.Struct(&cr3)
	cr3.PostId = postID
	cr3.ParentId = r1.Id
	id, err = s.r.Add(s.ctx, &cr3)
	cr3.Id = id
	s.NoError(err)
	expected = append(expected, &cr3)

	var cr11 models.CommentDTO
	gofakeit.Struct(&cr11)
	cr11.PostId = postID
	cr11.ParentId = cr1.Id
	id, err = s.r.Add(s.ctx, &cr11)
	cr11.Id = id
	s.NoError(err)
	expected = append(expected, &cr11)

	var cr12 models.CommentDTO
	gofakeit.Struct(&cr12)
	cr12.PostId = postID
	cr12.ParentId = cr1.Id
	id, err = s.r.Add(s.ctx, &cr12)
	cr12.Id = id
	s.NoError(err)
	expected = append(expected, &cr12)

	var cr21 models.CommentDTO
	gofakeit.Struct(&cr21)
	cr21.PostId = postID
	cr21.ParentId = cr2.Id
	id, err = s.r.Add(s.ctx, &cr21)
	cr21.Id = id
	s.NoError(err)
	expected = append(expected, &cr21)

	var r2 models.CommentDTO
	gofakeit.Struct(&r2)
	r2.PostId = postID
	r2.ParentId = 0
	id, err = s.r.Add(s.ctx, &r2)
	r2.Id = id
	s.NoError(err)
	expected = append(expected, &r2)

	var crr1 models.CommentDTO
	gofakeit.Struct(&crr1)
	crr1.PostId = postID
	crr1.ParentId = r2.Id
	id, err = s.r.Add(s.ctx, &crr1)
	crr1.Id = id
	s.NoError(err)
	expected = append(expected, &crr1)

	var crr2 models.CommentDTO
	gofakeit.Struct(&crr2)
	crr2.PostId = postID
	crr2.ParentId = r2.Id
	id, err = s.r.Add(s.ctx, &crr2)
	crr2.Id = id
	s.NoError(err)
	expected = append(expected, &crr2)

	var crr12 models.CommentDTO
	gofakeit.Struct(&crr12)
	crr12.PostId = postID
	crr12.ParentId = crr1.Id
	id, err = s.r.Add(s.ctx, &crr12)
	crr12.Id = id
	s.NoError(err)
	expected = append(expected, &crr12)

	// when
	actual, err := s.r.GetAllOfPost(s.ctx, postID, 5, 5)

	// then
	s.NoError(err)
	s.ElementsMatch(expected[5:10], actual)
}

func (s *PgCommentRepositoryTestSuite) TestGetEmptyComments() {

	// when
	actual, err := s.r.GetAllOfPost(s.ctx, 1, 0, 10)

	// then
	s.NoError(err)
	s.Len(actual, 0)
}

func (s *PgCommentRepositoryTestSuite) getRandomPostId() int64 {
	var postId int64
	err := s.pool.QueryRow(s.ctx, "SELECT id_p FROM Post limit 1").Scan(&postId)
	if err == pgx.ErrNoRows {
		return 0
	}
	s.NoError(err)
	return postId
}

func TestPgCommentRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(PgCommentRepositoryTestSuite))
}
