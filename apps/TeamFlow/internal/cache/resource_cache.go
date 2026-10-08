package cache

import (
	"context"
	"fmt"
	"time"

	"teamflow/internal/database"
	"teamflow/pkg/utils"
)

const (
	detailTTL    = 5 * time.Minute
	detailJitter = 5 * time.Minute
	listTTL      = time.Minute
	listJitter   = time.Minute
)

const cacheOpTimeout = 2 * time.Second

func ProjectKey(projectID uint) string { return fmt.Sprintf("project:%d", projectID) }
func TeamKey(teamID uint) string       { return fmt.Sprintf("team:%d", teamID) }
func TaskKey(taskID uint) string       { return fmt.Sprintf("task:%d", taskID) }

func TaskCommentsKey(taskID uint, page, size int) string {
	return fmt.Sprintf("task:comments:%d:page:%d:size:%d", taskID, page, size)
}

func ProjectListKey(teamID uint, page, size int) string {
	return fmt.Sprintf("project:list:team:%d:page:%d:size:%d", teamID, page, size)
}

func TeamListKey(userID uint, page, size int) string {
	return fmt.Sprintf("team:list:user:%d:page:%d:size:%d", userID, page, size)
}

func TaskListKey(projectID uint, page, size int) string {
	return fmt.Sprintf("task:list:project:%d:page:%d:size:%d", projectID, page, size)
}

func ProjectStatsKey(projectID uint) string { return fmt.Sprintf("project:stats:%d", projectID) }

// GetResourceCache reads a business-data cache. Redis outages degrade to a cache miss.
func GetResourceCache(key string, out any) (bool, error) {
	if database.RDB == nil {
		return false, nil
	}
	return utils.GetCache(key, out)
}

// SetDetailCache writes a detail cache with a 5–10 minute TTL.
func SetDetailCache(key string, value any) error {
	if database.RDB == nil {
		return nil
	}
	return utils.SetCacheWithJitter(key, value, detailTTL, detailJitter)
}

// SetListCache writes a list or aggregate cache with a 1–2 minute TTL.
func SetListCache(key string, value any) error {
	if database.RDB == nil {
		return nil
	}
	return utils.SetCacheWithJitter(key, value, listTTL, listJitter)
}

func DeleteResourceCache(keys ...string) error {
	if database.RDB == nil || len(keys) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
	defer cancel()
	return database.RDB.Del(ctx, keys...).Err()
}

// DeleteResourceCachePrefix removes all variants of a paginated cache after a write.
func DeleteResourceCachePrefix(prefix string) error {
	if database.RDB == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
	defer cancel()

	var cursor uint64
	for {
		keys, next, err := database.RDB.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := database.RDB.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		if next == 0 {
			return nil
		}
		cursor = next
	}
}

func InvalidateProject(projectID, teamID uint) error {
	if err := DeleteResourceCache(ProjectKey(projectID), ProjectStatsKey(projectID)); err != nil {
		return err
	}
	if err := DeleteResourceCachePrefix(fmt.Sprintf("task:list:project:%d:", projectID)); err != nil {
		return err
	}
	return DeleteResourceCachePrefix(fmt.Sprintf("project:list:team:%d:", teamID))
}

func InvalidateTeam(teamID uint) error {
	if err := DeleteResourceCache(TeamKey(teamID)); err != nil {
		return err
	}
	if err := DeleteResourceCachePrefix("team:list:"); err != nil {
		return err
	}
	return DeleteResourceCachePrefix(fmt.Sprintf("project:list:team:%d:", teamID))
}

func InvalidateTask(taskID, projectID uint) error {
	if err := DeleteResourceCache(TaskKey(taskID), ProjectStatsKey(projectID)); err != nil {
		return err
	}
	return DeleteResourceCachePrefix(fmt.Sprintf("task:list:project:%d:", projectID))
}

func InvalidateTaskComments(taskID uint) error {
	return DeleteResourceCachePrefix(fmt.Sprintf("task:comments:%d:", taskID))
}
