package inventory

import (
	"context"
	"testing"

	"github.com/docker/distribution"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/storage"
	"github.com/docker/distribution/registry/storage/driver"
	"github.com/docker/distribution/registry/storage/driver/inmemory"
	"github.com/docker/distribution/testutil"
	"github.com/stretchr/testify/require"
)

type env struct {
	ctx      context.Context
	driver   driver.StorageDriver
	registry distribution.Namespace
}

func newEnv(t *testing.T) *env {
	t.Helper()

	env := &env{
		ctx:    context.Background(),
		driver: inmemory.New(),
	}

	reg, err := storage.NewRegistry(env.ctx, env.driver)
	require.NoError(t, err)

	env.registry = reg

	return env
}

type repoTag struct {
	repoName string
	tags     []string
}

func TestIventory(t *testing.T) {
	var tests = []struct {
		name      string
		repoTags  []repoTag
		tagTotals bool
		expected  *Inventory
	}{
		{
			name: "single repository",
			repoTags: []repoTag{
				{
					repoName: "group/repo",
					tags: []string{
						"latest",
					},
				},
			},
			tagTotals: true,
			expected: &Inventory{
				Repositories: []Repository{
					{
						Path:     "group/repo",
						Group:    "group",
						TagCount: 1,
					},
				},
			},
		},
		{
			name: "single repository no tag totals",
			repoTags: []repoTag{
				{
					repoName: "group/repo",
					tags: []string{
						"latest",
					},
				},
			},
			tagTotals: false,
			expected: &Inventory{
				Repositories: []Repository{
					{
						Path:     "group/repo",
						Group:    "group",
						TagCount: 0,
					},
				},
			},
		},
		{
			name: "single repository no tags",
			repoTags: []repoTag{
				{
					repoName: "group/repo",
					tags:     []string{},
				},
			},
			tagTotals: true,
			expected: &Inventory{
				Repositories: []Repository{
					{
						Path:     "group/repo",
						Group:    "group",
						TagCount: 0,
					},
				},
			},
		},
		{
			name: "single repository no tags no tag totals",
			repoTags: []repoTag{
				{
					repoName: "group/repo",
					tags:     []string{},
				},
			},
			tagTotals: false,
			expected: &Inventory{
				Repositories: []Repository{
					{
						Path:     "group/repo",
						Group:    "group",
						TagCount: 0,
					},
				},
			},
		},
		{
			name: "single repository multiple tags",
			repoTags: []repoTag{
				{
					repoName: "group/repo",
					tags: []string{
						"latest",
						"v1.0.1",
						"v1.1.1",
						"v1.1.2",
						"v1.0.3",
						"latest", // retag
					},
				},
			},
			tagTotals: true,
			expected: &Inventory{
				Repositories: []Repository{
					{
						Path:     "group/repo",
						Group:    "group",
						TagCount: 5,
					},
				},
			},
		},
		{
			name: "multiple repository multiple tags",
			repoTags: []repoTag{
				{
					repoName: "group-1/repo-1",
					tags: []string{
						"latest",
						"v1.0.1",
					},
				},
				{
					repoName: "group-1/repo-2",
					tags: []string{
						"v1.0.1",
						"development",
					},
				},
				{
					repoName: "group-2/repo-1",
					tags: []string{
						"latest",
						"canary",
						"branch-slug-tag",
					},
				},
			},
			tagTotals: true,
			expected: &Inventory{
				Repositories: []Repository{
					{
						Path:     "group-1/repo-1",
						Group:    "group-1",
						TagCount: 2,
					},
					{
						Path:     "group-1/repo-2",
						Group:    "group-1",
						TagCount: 2,
					},
					{
						Path:     "group-2/repo-1",
						Group:    "group-2",
						TagCount: 3,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newEnv(t)

			for _, rt := range tt.repoTags {
				// Upload the manifest.
				n, err := reference.WithName(rt.repoName)
				require.NoError(t, err)

				repo, err := env.registry.Repository(env.ctx, n)
				require.NoError(t, err)

				img, err := testutil.UploadRandomSchema2Image(repo)
				require.NoError(t, err)

				// Tag the manifest.
				tagService := repo.Tags(env.ctx)

				for _, t := range rt.tags {
					tagService.Tag(env.ctx, t, distribution.Descriptor{Digest: img.ManifestDigest})
				}
			}

			it := NewTaker(env.registry, tt.tagTotals)
			iv, err := it.Run(env.ctx)
			require.NoError(t, err)

			require.ElementsMatch(t, tt.expected.Repositories, iv.Repositories)
		})
	}
}

func TestSummary(t *testing.T) {
	var tests = []struct {
		name     string
		iv       *Inventory
		expected *Summary
	}{
		{
			name: "single repository",
			iv: &Inventory{
				Repositories: []Repository{
					{
						Path:     "group-1/repo-1",
						Group:    "group-1",
						TagCount: 2,
					},
				},
			},
			expected: &Summary{
				Groups: []Group{
					{
						Name:            "group-1",
						TagCount:        2,
						RepositoryCount: 1,
					},
				},
			},
		},
		{
			name: "multiple repository",
			iv: &Inventory{
				Repositories: []Repository{
					{
						Path:     "group-1/repo-1",
						Group:    "group-1",
						TagCount: 2,
					},
					{
						Path:     "group-1/repo-2",
						Group:    "group-1",
						TagCount: 16,
					},
					{
						Path:     "group-1/repo-2/project",
						Group:    "group-1",
						TagCount: 3,
					},
					{
						Path:     "group-2/repo-1",
						Group:    "group-2",
						TagCount: 5,
					},
				},
			},
			expected: &Summary{
				Groups: []Group{
					{
						Name:            "group-1",
						TagCount:        21,
						RepositoryCount: 3,
					},
					{
						Name:            "group-2",
						TagCount:        5,
						RepositoryCount: 1,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.iv.Summary()

			require.ElementsMatch(t, tt.expected.Groups, s.Groups)
		})
	}
}
