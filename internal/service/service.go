package service

import "github.com/mi-raf/comment-project/internal/database"

type (
	PostService struct {
		r database.PostRepository
	}
)

func NewPostService(r database.PostRepository) *PostService {
	return &PostService{r}
}
