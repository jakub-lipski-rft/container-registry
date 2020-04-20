package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/ocischema"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/docker/distribution/registry/storage/driver"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

// Importer populates the registry database with filesystem metadata. This is only meant to be used for an initial
// one-off migration, starting with an empty database.
type Importer struct {
	storageDriver driver.StorageDriver
	registry      distribution.Namespace

	repositoryStore            *repositoryStore
	manifestConfigurationStore *manifestConfigurationStore
	manifestStore              *manifestStore
	manifestListStore          *manifestListStore
	tagStore                   *tagStore
	layerStore                 *layerStore
}

// NewImporter creates a new Importer.
func NewImporter(db Queryer, storageDriver driver.StorageDriver, registry distribution.Namespace) *Importer {
	return &Importer{
		storageDriver:              storageDriver,
		registry:                   registry,
		manifestConfigurationStore: NewManifestConfigurationStore(db),
		manifestStore:              NewManifestStore(db),
		manifestListStore:          NewManifestListStore(db),
		layerStore:                 NewLayerStore(db),
		repositoryStore:            NewRepositoryStore(db),
		tagStore:                   NewTagStore(db),
	}
}

// parseRepositoryPath parses a repository path and returns the corresponding repository name and its ancestors by
// splitting the path on forward slashes.
func parseRepositoryPath(path string) (string, []string) {
	segments := strings.Split(filepath.Clean(path), "/")
	name := segments[len(segments)-1]
	ancestors := segments[:len(segments)-1]

	return name, ancestors
}

func (imp *Importer) findOrCreateDBManifest(ctx context.Context, m *models.Manifest) (*models.Manifest, error) {
	dbManifest, err := imp.manifestStore.FindByDigest(ctx, m.Digest)
	if err != nil {
		return nil, fmt.Errorf("error searching for manifest: %w", err)
	}

	if dbManifest == nil {
		if err := imp.manifestStore.Create(ctx, m); err != nil {
			return nil, fmt.Errorf("error creating manifest: %w", err)
		}
		dbManifest = m
	}

	return dbManifest, nil
}

func (imp *Importer) findOrCreateDBLayer(ctx context.Context, fsRepo distribution.Repository, l *models.Layer) (*models.Layer, error) {
	dbLayer, err := imp.layerStore.FindByDigest(ctx, l.Digest)
	if err != nil {
		return nil, fmt.Errorf("error searching for layer: %w", err)
	}

	if dbLayer == nil {
		// v1 manifests don't include the layers blob size and media type, so we must Stat the blob to know
		if l.Size == 0 {
			blobStore := fsRepo.Blobs(ctx)
			desc, err := blobStore.Stat(ctx, digest.Digest(l.Digest))
			if err != nil {
				return nil, fmt.Errorf("error obtaining layer size: %w", err)
			}
			l.Size = desc.Size
			l.MediaType = desc.MediaType
		}

		if err := imp.layerStore.Create(ctx, l); err != nil {
			return nil, fmt.Errorf("error creating layer: %w", err)
		}
		dbLayer = l
	}

	return dbLayer, nil
}

func (imp *Importer) findOrCreateDBManifestConfig(ctx context.Context, mc *models.ManifestConfiguration) (*models.ManifestConfiguration, error) {
	dbConfig, err := imp.manifestConfigurationStore.FindByDigest(ctx, mc.Digest)
	if err != nil {
		return nil, fmt.Errorf("error searching for manifest configuration: %w", err)
	}

	if dbConfig == nil {
		if err := imp.manifestConfigurationStore.Create(ctx, mc); err != nil {
			return nil, fmt.Errorf("error creating manifest configuration: %w", err)
		}
		dbConfig = mc
	}

	return dbConfig, nil
}

func (imp *Importer) importLayer(ctx context.Context, fsRepo distribution.Repository, dbManifest *models.Manifest, l *models.Layer) error {
	dbLayer, err := imp.findOrCreateDBLayer(ctx, fsRepo, l)
	if err != nil {
		return err
	}
	if err := imp.manifestStore.AssociateLayer(ctx, dbManifest, dbLayer); err != nil {
		return fmt.Errorf("error associating layer with manifest: %w", err)
	}

	return nil
}

func (imp *Importer) importSchema1Layers(ctx context.Context, fsRepo distribution.Repository, dbManifest *models.Manifest, fsLayers []schema1.FSLayer) error {
	total := len(fsLayers)
	for i, fsLayer := range fsLayers {
		dgst := fsLayer.BlobSum.String()
		log := logrus.WithFields(logrus.Fields{
			"digest": dgst,
			"count":  i + 1,
			"total":  total,
		})
		log.Info("importing layer")

		if err := imp.importLayer(ctx, fsRepo, dbManifest, &models.Layer{Digest: dgst}); err != nil {
			log.WithError(err).Error("error importing layer")
			continue
		}
	}

	return nil
}

func (imp *Importer) importLayers(ctx context.Context, fsRepo distribution.Repository, dbManifest *models.Manifest, fsLayers []distribution.Descriptor) error {
	total := len(fsLayers)
	for i, fsLayer := range fsLayers {
		dgst := fsLayer.Digest.String()
		log := logrus.WithFields(logrus.Fields{
			"digest":     dgst,
			"media_type": fsLayer.MediaType,
			"size":       fsLayer.Size,
			"count":      i + 1,
			"total":      total,
		})
		log.Info("importing layer")

		err := imp.importLayer(ctx, fsRepo, dbManifest, &models.Layer{
			MediaType: fsLayer.MediaType,
			Digest:    dgst,
			Size:      fsLayer.Size,
		})
		if err != nil {
			log.WithError(err).Error("error importing layer")
			continue
		}
	}

	return nil
}

func (imp *Importer) importSchema1Manifest(ctx context.Context, fsRepo distribution.Repository, dbRepo *models.Repository, m *schema1.SignedManifest, dgst digest.Digest) (*models.Manifest, error) {
	// parse manifest payload
	_, payload, err := m.Payload()
	if err != nil {
		return nil, fmt.Errorf("error parsing manifest payload: %w", err)
	}

	// find or create DB manifest
	dbManifest, err := imp.findOrCreateDBManifest(ctx, &models.Manifest{
		SchemaVersion: m.SchemaVersion,
		MediaType:     schema1.MediaTypeSignedManifest,
		Digest:        dgst.String(),
		Payload:       payload,
	})
	if err != nil {
		return nil, err
	}

	// import manifest layers
	if err := imp.importSchema1Layers(ctx, fsRepo, dbManifest, m.FSLayers); err != nil {
		return nil, fmt.Errorf("error importing layers: %w", err)
	}

	// associate manifest with repository
	if err := imp.repositoryStore.AssociateManifest(ctx, dbRepo, dbManifest); err != nil {
		return nil, fmt.Errorf("error associating manifest with repository: %w", err)
	}

	return dbManifest, nil
}

func (imp *Importer) importSchema2Manifest(ctx context.Context, fsRepo distribution.Repository, dbRepo *models.Repository, m *schema2.DeserializedManifest, dgst digest.Digest) (*models.Manifest, error) {
	blobStore := fsRepo.Blobs(ctx)
	configPayload, err := blobStore.Get(ctx, m.Config.Digest)
	if err != nil {
		return nil, fmt.Errorf("error obtaining manifest configuration payload: %w", err)
	}

	// find or create DB manifest configuration
	dbConfig, err := imp.findOrCreateDBManifestConfig(ctx, &models.ManifestConfiguration{
		MediaType: m.Config.MediaType,
		Digest:    m.Config.Digest.String(),
		Size:      m.Config.Size,
		Payload:   configPayload,
	})
	if err != nil {
		return nil, err
	}

	_, payload, err := m.Payload()
	if err != nil {
		return nil, fmt.Errorf("error parsing manifest payload: %w", err)
	}

	// find or create DB manifest
	dbManifest, err := imp.findOrCreateDBManifest(ctx, &models.Manifest{
		SchemaVersion: m.SchemaVersion,
		MediaType:     m.MediaType,
		Digest:        dgst.String(),
		ConfigurationID: sql.NullInt64{
			Int64: int64(dbConfig.ID),
			Valid: true,
		},
		Payload: payload,
	})
	if err != nil {
		return nil, err
	}

	// import manifest layers
	if err := imp.importLayers(ctx, fsRepo, dbManifest, m.Layers); err != nil {
		return nil, fmt.Errorf("error importing layers: %w", err)
	}

	// associate repository with manifest
	if err := imp.repositoryStore.AssociateManifest(ctx, dbRepo, dbManifest); err != nil {
		return nil, fmt.Errorf("error associating manifest with repository: %w", err)
	}

	return dbManifest, nil
}

func (imp *Importer) importOCIManifest(ctx context.Context, fsRepo distribution.Repository, dbRepo *models.Repository, m *ocischema.DeserializedManifest, dgst digest.Digest) (*models.Manifest, error) {
	blobStore := fsRepo.Blobs(ctx)
	configPayload, err := blobStore.Get(ctx, m.Config.Digest)
	if err != nil {
		return nil, fmt.Errorf("error obtaining manifest configuration payload: %w", err)
	}

	// find or create DB manifest configuration
	dbConfig, err := imp.findOrCreateDBManifestConfig(ctx, &models.ManifestConfiguration{
		MediaType: m.Config.MediaType,
		Digest:    m.Config.Digest.String(),
		Size:      m.Config.Size,
		Payload:   configPayload,
	})
	if err != nil {
		return nil, err
	}

	_, payload, err := m.Payload()
	if err != nil {
		return nil, fmt.Errorf("error parsing manifest payload: %w", err)
	}

	// find or create DB manifest
	dbManifest, err := imp.findOrCreateDBManifest(ctx, &models.Manifest{
		SchemaVersion: m.SchemaVersion,
		MediaType:     v1.MediaTypeImageManifest,
		Digest:        dgst.String(),
		ConfigurationID: sql.NullInt64{
			Int64: int64(dbConfig.ID),
			Valid: true,
		},
		Payload: payload,
	})
	if err != nil {
		return nil, err
	}

	// import manifest layers
	if err := imp.importLayers(ctx, fsRepo, dbManifest, m.Layers); err != nil {
		return nil, fmt.Errorf("error importing layers: %w", err)
	}

	// associate repository with manifest
	if err := imp.repositoryStore.AssociateManifest(ctx, dbRepo, dbManifest); err != nil {
		return nil, fmt.Errorf("error associating manifest with repository: %w", err)
	}

	return dbManifest, nil
}

func (imp *Importer) importManifestList(ctx context.Context, fsRepo distribution.Repository, dbRepo *models.Repository, ml *manifestlist.DeserializedManifestList, dgst digest.Digest) (*models.ManifestList, error) {
	_, payload, err := ml.Payload()
	if err != nil {
		return nil, fmt.Errorf("error parsing manifest list payload: %w", err)
	}

	// media type can be either Docker (`application/vnd.docker.distribution.manifest.list.v2+json`) or OCI (empty)
	mediaType := sql.NullString{String: ml.MediaType, Valid: len(ml.MediaType) > 0}

	// create manifest list on DB
	dbManifestList := &models.ManifestList{
		SchemaVersion: ml.SchemaVersion,
		MediaType:     mediaType,
		Digest:        dgst.String(),
		Payload:       payload,
	}
	if err := imp.manifestListStore.Create(ctx, dbManifestList); err != nil {
		return nil, fmt.Errorf("error creating manifest list: %w", err)
	}

	manifestService, err := fsRepo.Manifests(ctx)
	if err != nil {
		return nil, fmt.Errorf("error constructing manifest service: %w", err)
	}

	// import manifests in list
	total := len(ml.Manifests)
	for i, m := range ml.Manifests {
		fsManifest, err := manifestService.Get(ctx, m.Digest)
		if err != nil {
			logrus.WithError(err).Error("error retrieving manifest")
			continue
		}

		logrus.WithFields(logrus.Fields{
			"digest": m.Digest.String(),
			"count":  i + 1,
			"total":  total,
			"type":   fmt.Sprintf("%T", fsManifest),
		}).Info("importing manifest from list")

		dbManifest, err := imp.importManifest(ctx, fsRepo, dbRepo, fsManifest, m.Digest)
		if err != nil {
			logrus.WithError(err).Error("error importing manifest")
			continue
		}

		if err := imp.manifestListStore.AssociateManifest(ctx, dbManifestList, dbManifest); err != nil {
			logrus.WithError(err).Error("error associating manifest and manifest list")
			continue
		}
	}

	// associate repository and manifest list
	if err := imp.repositoryStore.AssociateManifestList(ctx, dbRepo, dbManifestList); err != nil {
		return nil, fmt.Errorf("error associating repository and manifest list: %w", err)
	}

	return dbManifestList, nil
}

func (imp *Importer) importManifest(ctx context.Context, fsRepo distribution.Repository, dbRepo *models.Repository, m distribution.Manifest, dgst digest.Digest) (*models.Manifest, error) {
	switch fsManifest := m.(type) {
	case *schema1.SignedManifest:
		return imp.importSchema1Manifest(ctx, fsRepo, dbRepo, fsManifest, dgst)
	case *schema2.DeserializedManifest:
		return imp.importSchema2Manifest(ctx, fsRepo, dbRepo, fsManifest, dgst)
	case *ocischema.DeserializedManifest:
		return imp.importOCIManifest(ctx, fsRepo, dbRepo, fsManifest, dgst)
	default:
		return nil, fmt.Errorf("unknown manifest class: %T", fsManifest)
	}
}

func (imp *Importer) importManifests(ctx context.Context, fsRepo distribution.Repository, dbRepo *models.Repository) error {
	manifestService, err := fsRepo.Manifests(ctx)
	if err != nil {
		return fmt.Errorf("error constructing manifest service: %w", err)
	}
	manifestEnumerator, ok := manifestService.(distribution.ManifestEnumerator)
	if !ok {
		return fmt.Errorf("error converting ManifestService into ManifestEnumerator")
	}

	index := 0
	err = manifestEnumerator.Enumerate(ctx, func(dgst digest.Digest) error {
		index++

		m, err := manifestService.Get(ctx, dgst)
		if err != nil {
			return fmt.Errorf("error retrieving manifest %q: %w", dgst, err)
		}

		log := logrus.WithFields(logrus.Fields{"digest": dgst, "count": index, "type": fmt.Sprintf("%T", m)})

		switch fsManifest := m.(type) {
		case *manifestlist.DeserializedManifestList:
			log.Info("importing manifest list")
			_, err = imp.importManifestList(ctx, fsRepo, dbRepo, fsManifest, dgst)
		default:
			log.Info("importing manifest")
			_, err = imp.importManifest(ctx, fsRepo, dbRepo, fsManifest, dgst)
		}

		return err
	})

	return err
}

func (imp *Importer) importTags(ctx context.Context, fsRepo distribution.Repository, dbRepo *models.Repository) error {
	tagService := fsRepo.Tags(ctx)
	fsTags, err := tagService.All(ctx)
	if err != nil {
		return fmt.Errorf("error reading tags: %w", err)
	}

	// sort tags to ensure reproducible imports across different OSs using the filesystem storage driver
	// see https://gitlab.com/gitlab-org/container-registry/-/issues/88
	sort.Strings(fsTags)

	total := len(fsTags)
	for i, fsTag := range fsTags {
		log := logrus.WithFields(logrus.Fields{"name": fsTag, "count": i + 0, "total": total})

		// read tag details from the filesystem
		desc, err := tagService.Get(ctx, fsTag)
		if err != nil {
			log.WithError(err).Error("error reading tag details")
			continue
		}

		dgst := desc.Digest.String()
		log = log.WithField("target", dgst)
		log.Info("importing tag")

		dbTag := &models.Tag{Name: fsTag, RepositoryID: dbRepo.ID}

		// find corresponding manifest or manifest list in DB
		dbManifest, err := imp.manifestStore.FindByDigest(ctx, dgst)
		if err != nil {
			return fmt.Errorf("error finding target manifest: %w", err)
		}
		if dbManifest == nil {
			dbManifestList, err := imp.manifestListStore.FindByDigest(ctx, dgst)
			if err != nil {
				return fmt.Errorf("error finding target manifest list: %w", err)
			}
			if dbManifestList == nil {
				log.WithError(err).Errorf("no manifest or manifest list found for digest %q", dgst)
				continue
			}
			dbTag.ManifestListID = sql.NullInt64{Int64: int64(dbManifestList.ID), Valid: true}
		} else {
			dbTag.ManifestID = sql.NullInt64{Int64: int64(dbManifest.ID), Valid: true}
		}

		// create tag
		if err := imp.tagStore.Create(ctx, dbTag); err != nil {
			log.WithError(err).Error("error creating tag")
			continue
		}
	}

	return nil
}

// importParentRepositories creates parent repositories recursively, preserving their hierarchical relationship.
// Returns the direct parent repository.
func (imp *Importer) importParentRepositories(ctx context.Context, path string) (*models.Repository, error) {
	name, parents := parseRepositoryPath(path)

	logrus.WithField("path", path).Info("importing repository parents")

	// create indirect parents
	var currParentID int
	for _, parent := range parents {
		// check if already exists
		dbRepo, err := imp.repositoryStore.FindByPath(ctx, parent)
		if err != nil {
			return nil, fmt.Errorf("error finding parent repository: %w", err)
		}
		if dbRepo == nil {
			logrus.WithField("path", parent).Info("creating indirect parent")
			name, _ := parseRepositoryPath(parent)
			dbRepo = &models.Repository{
				Name: name,
				Path: parent,
				ParentID: sql.NullInt64{
					Int64: int64(currParentID),
					Valid: currParentID > 0,
				},
			}
			if err := imp.repositoryStore.Create(ctx, dbRepo); err != nil {
				return nil, fmt.Errorf("error creating parent repository: %w", err)
			}
		}

		currParentID = dbRepo.ID
	}

	// create direct parent
	dbParent := &models.Repository{
		Name: name,
		Path: path,
		ParentID: sql.NullInt64{
			Int64: int64(currParentID),
			Valid: currParentID > 0,
		},
	}
	if err := imp.repositoryStore.Create(ctx, dbParent); err != nil {
		return nil, fmt.Errorf("error creating parent repository: %w", err)
	}

	return dbParent, nil
}

func (imp *Importer) importRepository(ctx context.Context, path string) error {
	named, err := reference.WithName(path)
	if err != nil {
		return fmt.Errorf("error parsing repository name: %w", err)
	}
	fsRepo, err := imp.registry.Repository(ctx, named)
	if err != nil {
		return fmt.Errorf("error constructing repository: %w", err)
	}

	// break repository path in name and parents
	name, parents := parseRepositoryPath(path)
	parentPath := filepath.Join(parents...)

	// find parent repository (if applicable)
	parentID := sql.NullInt64{}
	if parentPath != "" {
		parent, err := imp.repositoryStore.FindByPath(ctx, parentPath)
		if err != nil {
			return fmt.Errorf("error finding parent repository: %w", err)
		}
		if parent == nil {
			parent, err = imp.importParentRepositories(ctx, parentPath)
			if err != nil {
				return fmt.Errorf("error importing parent repositories: %w", err)
			}
		}
		parentID.Int64 = int64(parent.ID)
		parentID.Valid = true
	}

	// create repository
	dbRepo := &models.Repository{Name: name, Path: path, ParentID: parentID}
	if err := imp.repositoryStore.Create(ctx, dbRepo); err != nil {
		return fmt.Errorf("error creating repository: %w", err)
	}

	// import repository manifests
	if err := imp.importManifests(ctx, fsRepo, dbRepo); err != nil {
		return fmt.Errorf("error importing manifests: %w", err)
	}
	//import repository tags
	if err := imp.importTags(ctx, fsRepo, dbRepo); err != nil {
		return fmt.Errorf("error importing tags: %w", err)
	}

	return nil
}

func (imp *Importer) countRows(ctx context.Context) (map[string]int, error) {
	numRepositories, err := imp.repositoryStore.Count(ctx)
	if err != nil {
		return nil, err
	}
	numManifests, err := imp.manifestStore.Count(ctx)
	if err != nil {
		return nil, err
	}
	numManifestLists, err := imp.manifestListStore.Count(ctx)
	if err != nil {
		return nil, err
	}
	numManifestConfigs, err := imp.manifestConfigurationStore.Count(ctx)
	if err != nil {
		return nil, err
	}
	numLayers, err := imp.layerStore.Count(ctx)
	if err != nil {
		return nil, err
	}
	numTags, err := imp.tagStore.Count(ctx)
	if err != nil {
		return nil, err
	}

	count := map[string]int{
		"repositories":            numRepositories,
		"manifests":               numManifests,
		"manifest_lists":          numManifestLists,
		"manifest_configurations": numManifestConfigs,
		"layers":                  numLayers,
		"tags":                    numTags,
	}

	return count, nil
}

func (imp *Importer) isDatabaseEmpty(ctx context.Context) (bool, error) {
	counters, err := imp.countRows(ctx)
	if err != nil {
		return false, err
	}

	for _, c := range counters {
		if c > 0 {
			return false, nil
		}
	}

	return true, nil
}

// Import populates the registry database based on the metadata from the storage backend.
func (imp *Importer) Import(ctx context.Context) error {
	start := time.Now()
	logrus.Info("starting metadata import")

	empty, err := imp.isDatabaseEmpty(ctx)
	if err != nil {
		return fmt.Errorf("error checking if database is empty: %w", err)
	}
	if !empty {
		return errors.New("non-empty database")
	}

	repositoryEnumerator, ok := imp.registry.(distribution.RepositoryEnumerator)
	if !ok {
		return errors.New("error building repository enumerator")
	}

	index := 0
	err = repositoryEnumerator.Enumerate(ctx, func(path string) error {
		index++
		repoStart := time.Now()
		log := logrus.WithFields(logrus.Fields{"path": path, "count": index})
		log.Info("importing repository")

		if err := imp.importRepository(ctx, path); err != nil {
			log.WithError(err).Error("error importing repository")
			// if the storage driver failed to find a repository path (usually due to missing `_manifests/revisions`
			// or `_manifests/tags` folders) continue to the next one, otherwise stop as the error is unknown.
			if !errors.As(err, &driver.PathNotFoundError{}) {
				return err
			}
		}
		repoEnd := time.Since(repoStart).Seconds()
		log.WithField("duration_s", repoEnd).Info("import complete")

		return nil
	})
	if err != nil {
		return err
	}

	counters, err := imp.countRows(ctx)
	if err != nil {
		logrus.WithError(err).Error("error counting table rows")
	}

	logCounters := make(map[string]interface{}, len(counters))
	for t, n := range counters {
		logCounters[t] = n
	}

	t := time.Since(start).Seconds()
	logrus.WithField("duration_s", t).WithFields(logCounters).Info("metadata import complete")

	return err
}
