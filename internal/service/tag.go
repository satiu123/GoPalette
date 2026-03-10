package service

import (
	"context"
	"errors"

	"github.com/satiu123/GoPalette/internal/model"
	"github.com/satiu123/GoPalette/internal/repository"
)

type TagService struct {
	tagRepo repository.TagRepository
}

func NewTagService(tagRepo repository.TagRepository) *TagService {
	return &TagService{tagRepo: tagRepo}
}

func (s *TagService) CreateTag(ctx context.Context, name string) (*model.Tag, error) {
	tag := &model.Tag{Name: name}
	if err := s.tagRepo.Create(ctx, tag); err != nil {
		return nil, err
	}
	return tag, nil
}

func (s *TagService) ListTags(ctx context.Context) ([]model.Tag, error) {
	return s.tagRepo.FindAll(ctx)
}

func (s *TagService) DeleteTag(ctx context.Context, id int64) error {
	tag, err := s.tagRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if tag == nil {
		return errors.New("标签不存在")
	}
	return s.tagRepo.Delete(ctx, id)
}
