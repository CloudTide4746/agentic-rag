package cache

import (
	"testing"
)

func TestResourceCacheKeysAreScopedByAllQueryInputs(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"project detail", ProjectKey(12), "project:12"},
		{"team detail", TeamKey(7), "team:7"},
		{"task detail", TaskKey(99), "task:99"},
		{"task comments", TaskCommentsKey(99, 2, 20), "task:comments:99:page:2:size:20"},
		{"project list", ProjectListKey(7, 3, 10), "project:list:team:7:page:3:size:10"},
		{"team list", TeamListKey(5, 1, 10), "team:list:user:5:page:1:size:10"},
		{"task list", TaskListKey(12, 4, 25), "task:list:project:12:page:4:size:25"},
		{"project stats", ProjectStatsKey(12), "project:stats:12"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("key = %q, want %q", tt.got, tt.want)
			}
		})
	}
}
