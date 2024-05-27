package models

import "time"

type (
	PostDTO struct {
		Id            int64
		Author        string
		Title         string
		Text          string
		IsCommentable bool
		Time          time.Time `fake:"{futuredate}"`
	}

	CommentDTO struct {
		Id       int64
		PostId   int64
		ParentId int64
		Author   string
		Text     string
		Time     time.Time `fake:"{futuredate}"`
		Level    int
	}
)
