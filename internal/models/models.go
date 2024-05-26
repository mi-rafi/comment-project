package models

type (
	PostDTO struct {
		Id         int32
		Author     string
		Title      string
		Text       string
		IsComments bool
	}

	CommentDTO struct {
		Id       int32
		PostId   int32
		ParentId int32
		Author   string
		Text     string
	}
)
