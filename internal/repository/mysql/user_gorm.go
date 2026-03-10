package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/satiu123/GoPalette/internal/model"
	"gorm.io/gorm"
)

type userGromRepository struct {
	db  *gorm.DB
	rdb *redis.Client
}

const (
	userCacheKeyPrefix = "user:"
	userCacheTTL       = time.Hour * 1
)

func NewUserGormRepository(db *gorm.DB, rdb *redis.Client) *userGromRepository {
	return &userGromRepository{
		db:  db,
		rdb: rdb,
	}
}

func (r *userGromRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userGromRepository) FindByID(ctx context.Context, id int64) (*model.User, error) {
	cacheKey := userCacheKeyPrefix + "userid:" + strconv.FormatInt(id, 10)
	var user model.User
	// 先查 redis 缓存
	val, err := r.rdb.Get(ctx, cacheKey).Result()
	// 如果缓存命中
	if err == nil {
		if err := json.Unmarshal([]byte(val), &user); err == nil {
			return &user, nil
		}
	} else if err != redis.Nil {
		// 如果查询 Redis 出错且不是缓存未命中，返回错误
		return nil, err
	}

	// 缓存未命中或解析失败，从数据库查询
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 用户不存在，设置空值缓存并返回 nil
			r.rdb.Set(ctx, cacheKey, "", time.Minute*5) // 设置空值表示用户不存在
			return nil, errors.New("user not found")
		}
	}

	// 查询成功，设置缓存
	userJSON, err := json.Marshal(user)
	if err == nil {
		r.rdb.Set(ctx, cacheKey, userJSON, userCacheTTL)
	}

	return &user, nil
}

func (r *userGromRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	cacheKey := userCacheKeyPrefix + "username:" + username
	var user model.User
	// 先查 redis 缓存
	val, err := r.rdb.Get(ctx, cacheKey).Result()
	// 如果缓存命中
	if err == nil {
		if err := json.Unmarshal([]byte(val), &user); err == nil {
			return &user, nil
		}
	} else if err != redis.Nil {
		// 如果查询 Redis 出错且不是缓存未命中，返回错误
		return nil, err
	}
	// 缓存未命中或解析失败，从数据库查询
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 用户不存在，设置空值缓存并返回 nil
			r.rdb.Set(ctx, cacheKey, "", time.Minute*5) // 设置空值表示用户不存在
			return nil, errors.New("user not found")
		}
	}

	// 查询成功，设置缓存
	userJSON, err := json.Marshal(user)
	if err == nil {
		r.rdb.Set(ctx, cacheKey, userJSON, userCacheTTL)
	}

	return &user, nil
}

func (r *userGromRepository) Update(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return err
	}
	// 更新成功后，删除相关缓存
	cacheKeyByID := userCacheKeyPrefix + "userid:" + strconv.FormatInt(user.ID, 10)
	cacheKeyByUsername := userCacheKeyPrefix + "username:" + user.Username
	r.rdb.Del(ctx, cacheKeyByID, cacheKeyByUsername)
	return nil

}
