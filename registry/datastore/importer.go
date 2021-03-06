package datastore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest"
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
	registry            distribution.Namespace
	blobTransferService distribution.BlobTransferService
	db                  *DB
	repositoryStore     RepositoryStore
	manifestStore       ManifestStore
	tagStore            TagStore
	blobStore           BlobStore

	importDanglingManifests bool
	importDanglingBlobs     bool
	requireEmptyDatabase    bool
	dryRun                  bool
}

// ImporterOption provides functional options for the Importer.
type ImporterOption func(*Importer)

// WithImportDanglingManifests configures the Importer to import all manifests
// rather than only tagged manifests.
func WithImportDanglingManifests(imp *Importer) {
	imp.importDanglingManifests = true
}

// WithImportDanglingBlobs configures the Importer to import all blobs
// rather than only blobs referenced by manifests.
func WithImportDanglingBlobs(imp *Importer) {
	imp.importDanglingBlobs = true
}

// WithRequireEmptyDatabase configures the Importer to stop import unless the
// database being imported to is empty.
func WithRequireEmptyDatabase(imp *Importer) {
	imp.requireEmptyDatabase = true
}

// WithDryRun configures the Importer to use a single transacton which is rolled
// back and the end of an import cycle.
func WithDryRun(imp *Importer) {
	imp.dryRun = true
}

// WithBlobTransferService configures the Importer to use the passed BlobTransferService.
func WithBlobTransferService(bts distribution.BlobTransferService) ImporterOption {
	return func(imp *Importer) {
		imp.blobTransferService = bts
	}
}

// NewImporter creates a new Importer.
func NewImporter(db *DB, registry distribution.Namespace, opts ...ImporterOption) *Importer {
	imp := &Importer{
		registry: registry,
		db:       db,
	}

	for _, o := range opts {
		o(imp)
	}

	imp.loadStores(imp.db)

	return imp
}

func (imp *Importer) beginTx(ctx context.Context) (Transactor, error) {
	tx, err := imp.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	imp.loadStores(tx)

	return tx, nil
}

func (imp *Importer) loadStores(db Queryer) {
	imp.manifestStore = NewManifestStore(db)
	imp.blobStore = NewBlobStore(db)
	imp.repositoryStore = NewRepositoryStore(db)
	imp.tagStore = NewTagStore(db)
}

func (imp *Importer) findOrCreateDBManifest(ctx context.Context, dbRepo *models.Repository, m *models.Manifest) (*models.Manifest, error) {
	dbManifest, err := imp.repositoryStore.FindManifestByDigest(ctx, dbRepo, m.Digest)
	if err != nil {
		return nil, fmt.Errorf("searching for manifest: %w", err)
	}

	if dbManifest == nil {
		if err := imp.manifestStore.Create(ctx, m); err != nil {
			return nil, fmt.Errorf("creating manifest: %w", err)
		}
		dbManifest = m
	}

	return dbManifest, nil
}

func (imp *Importer) importLayer(ctx context.Context, dbRepo *models.Repository, dbManifest *models.Manifest, dbLayer *models.Blob) error {
	if err := imp.blobStore.CreateOrFind(ctx, dbLayer); err != nil {
		return fmt.Errorf("creating layer blob: %w", err)
	}

	if err := imp.manifestStore.AssociateLayerBlob(ctx, dbManifest, dbLayer); err != nil {
		return fmt.Errorf("associating layer blob with manifest: %w", err)
	}

	if err := imp.repositoryStore.LinkBlob(ctx, dbRepo, dbLayer.Digest); err != nil {
		return fmt.Errorf("linking layer blob to repository: %w", err)
	}

	if err := imp.transferBlob(ctx, dbLayer.Digest); err != nil {
		return fmt.Errorf("transferring layer blob: %w", err)
	}

	return nil
}

func (imp *Importer) importLayers(ctx context.Context, dbRepo *models.Repository, dbManifest *models.Manifest, fsLayers []distribution.Descriptor) error {
	total := len(fsLayers)
	for i, fsLayer := range fsLayers {
		log := logrus.WithFields(logrus.Fields{
			"digest":     fsLayer.Digest,
			"media_type": fsLayer.MediaType,
			"size":       fsLayer.Size,
			"count":      i + 1,
			"total":      total,
		})
		log.Info("importing layer")

		err := imp.importLayer(ctx, dbRepo, dbManifest, &models.Blob{
			MediaType: fsLayer.MediaType,
			Digest:    fsLayer.Digest,
			Size:      fsLayer.Size,
		})
		if err != nil {
			log.WithError(err).Error("importing layer")
			continue
		}
	}

	return nil
}

func (imp *Importer) transferBlob(ctx context.Context, d digest.Digest) error {
	if imp.dryRun || imp.blobTransferService == nil {
		return nil
	}

	start := time.Now()
	if err := imp.blobTransferService.Transfer(ctx, d); err != nil {
		return err
	}

	end := time.Since(start).Seconds()
	logrus.WithFields(logrus.Fields{
		"digest":     d,
		"duration_s": end,
	}).Info("blob transfer complete")

	return nil
}

type v2Manifest interface {
	distribution.Manifest
	version() manifest.Versioned
	config() distribution.Descriptor
	layers() []distribution.Descriptor
}

type schema2Extended struct {
	*schema2.DeserializedManifest
}

func (m *schema2Extended) version() manifest.Versioned       { return m.Versioned }
func (m *schema2Extended) config() distribution.Descriptor   { return m.Config }
func (m *schema2Extended) layers() []distribution.Descriptor { return m.Layers }

type ociExtended struct {
	*ocischema.DeserializedManifest
}

func (m *ociExtended) version() manifest.Versioned {
	// Helm chart manifests do not include a mediatype, set them to oci.
	if m.Versioned.MediaType == "" {
		m.Versioned.MediaType = v1.MediaTypeImageManifest
	}

	return m.Versioned
}

func (m *ociExtended) config() distribution.Descriptor   { return m.Config }
func (m *ociExtended) layers() []distribution.Descriptor { return m.Layers }

func (imp *Importer) importV2Manifest(ctx context.Context, fsRepo distribution.Repository, dbRepo *models.Repository, m v2Manifest, dgst digest.Digest) (*models.Manifest, error) {
	_, payload, err := m.Payload()
	if err != nil {
		return nil, fmt.Errorf("error parsing manifest payload: %w", err)
	}

	// get configuration blob payload
	blobStore := fsRepo.Blobs(ctx)
	configPayload, err := blobStore.Get(ctx, m.config().Digest)
	if err != nil {
		return nil, fmt.Errorf("error obtaining configuration payload: %w", err)
	}

	dbConfigBlob := &models.Blob{
		MediaType: m.config().MediaType,
		Digest:    m.config().Digest,
		Size:      m.config().Size,
	}
	if err = imp.blobStore.CreateOrFind(ctx, dbConfigBlob); err != nil {
		return nil, err
	}

	if err = imp.transferBlob(ctx, m.config().Digest); err != nil {
		return nil, fmt.Errorf("transferring config blob: %w", err)
	}

	// link configuration to repository
	if err := imp.repositoryStore.LinkBlob(ctx, dbRepo, dbConfigBlob.Digest); err != nil {
		return nil, fmt.Errorf("error associating configuration blob with repository: %w", err)
	}

	// find or create DB manifest
	dbManifest, err := imp.findOrCreateDBManifest(ctx, dbRepo, &models.Manifest{
		NamespaceID:   dbRepo.NamespaceID,
		RepositoryID:  dbRepo.ID,
		SchemaVersion: m.version().SchemaVersion,
		MediaType:     m.version().MediaType,
		Digest:        dgst,
		Payload:       payload,
		Configuration: &models.Configuration{
			MediaType: dbConfigBlob.MediaType,
			Digest:    dbConfigBlob.Digest,
			Payload:   configPayload,
		},
	})
	if err != nil {
		return nil, err
	}

	// import manifest layers
	if err := imp.importLayers(ctx, dbRepo, dbManifest, m.layers()); err != nil {
		return nil, fmt.Errorf("error importing layers: %w", err)
	}

	return dbManifest, nil
}

func (imp *Importer) importManifestList(ctx context.Context, fsRepo distribution.Repository, dbRepo *models.Repository, ml *manifestlist.DeserializedManifestList, dgst digest.Digest) (*models.Manifest, error) {
	_, payload, err := ml.Payload()
	if err != nil {
		return nil, fmt.Errorf("error parsing manifest list payload: %w", err)
	}

	// Media type can be either Docker (`application/vnd.docker.distribution.manifest.list.v2+json`) or OCI (empty).
	// We need to make it explicit if empty, otherwise we're not able to distinguish between media types.
	mediaType := ml.MediaType
	if mediaType == "" {
		mediaType = v1.MediaTypeImageIndex
	}

	// create manifest list on DB
	dbManifestList, err := imp.findOrCreateDBManifest(ctx, dbRepo, &models.Manifest{
		NamespaceID:   dbRepo.NamespaceID,
		RepositoryID:  dbRepo.ID,
		SchemaVersion: ml.SchemaVersion,
		MediaType:     mediaType,
		Digest:        dgst,
		Payload:       payload,
	})
	if err != nil {
		return nil, fmt.Errorf("creating manifest list: %w", err)
	}

	manifestService, err := fsRepo.Manifests(ctx)
	if err != nil {
		return nil, fmt.Errorf("constructing manifest service: %w", err)
	}

	// import manifests in list
	total := len(ml.Manifests)
	for i, m := range ml.Manifests {
		fsManifest, err := manifestService.Get(ctx, m.Digest)
		if err != nil {
			logrus.WithError(err).Error("retrieving manifest")
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
			logrus.WithError(err).Error("importing manifest")
			continue
		}

		if err := imp.manifestStore.AssociateManifest(ctx, dbManifestList, dbManifest); err != nil {
			logrus.WithError(err).Error("associating manifest and manifest list")
			continue
		}
	}

	return dbManifestList, nil
}

func (imp *Importer) importManifest(ctx context.Context, fsRepo distribution.Repository, dbRepo *models.Repository, m distribution.Manifest, dgst digest.Digest) (*models.Manifest, error) {
	switch fsManifest := m.(type) {
	case *schema1.SignedManifest:
		return nil, distribution.ErrSchemaV1Unsupported
	case *schema2.DeserializedManifest:
		return imp.importV2Manifest(ctx, fsRepo, dbRepo, &schema2Extended{fsManifest}, dgst)
	case *ocischema.DeserializedManifest:
		return imp.importV2Manifest(ctx, fsRepo, dbRepo, &ociExtended{fsManifest}, dgst)
	default:
		return nil, fmt.Errorf("unknown manifest class")
	}
}

func (imp *Importer) importManifests(ctx context.Context, fsRepo distribution.Repository, dbRepo *models.Repository) error {
	manifestService, err := fsRepo.Manifests(ctx)
	if err != nil {
		return fmt.Errorf("constructing manifest service: %w", err)
	}
	manifestEnumerator, ok := manifestService.(distribution.ManifestEnumerator)
	if !ok {
		return fmt.Errorf("converting ManifestService into ManifestEnumerator")
	}

	index := 0
	err = manifestEnumerator.Enumerate(ctx, func(dgst digest.Digest) error {
		index++

		m, err := manifestService.Get(ctx, dgst)
		if err != nil {
			return fmt.Errorf("retrieving manifest %q: %w", dgst, err)
		}

		log := logrus.WithFields(logrus.Fields{"digest": dgst, "count": index, "type": fmt.Sprintf("%T", m)})

		switch fsManifest := m.(type) {
		case *manifestlist.DeserializedManifestList:
			log.Info("importing manifest list")
			_, err = imp.importManifestList(ctx, fsRepo, dbRepo, fsManifest, dgst)
		default:
			log.Info("importing manifest")
			_, err = imp.importManifest(ctx, fsRepo, dbRepo, fsManifest, dgst)
			if errors.Is(err, distribution.ErrSchemaV1Unsupported) {
				logrus.WithError(err).Error("importing manifest")
				return nil
			}
		}

		return err
	})

	return err
}

func (imp *Importer) importTags(ctx context.Context, fsRepo distribution.Repository, dbRepo *models.Repository) error {
	manifestService, err := fsRepo.Manifests(ctx)
	if err != nil {
		return fmt.Errorf("constructing manifest service: %w", err)
	}

	tagService := fsRepo.Tags(ctx)
	fsTags, err := tagService.All(ctx)
	if err != nil {
		return fmt.Errorf("reading tags: %w", err)
	}

	total := len(fsTags)

	for i, fsTag := range fsTags {
		log := logrus.WithFields(logrus.Fields{"name": fsTag, "count": i + 1, "total": total})

		// read tag details from the filesystem
		desc, err := tagService.Get(ctx, fsTag)
		if err != nil {
			log.WithError(err).Error("reading tag details")
			continue
		}

		log = log.WithField("target", desc.Digest)
		log.Info("importing tag")

		dbTag := &models.Tag{Name: fsTag, NamespaceID: dbRepo.NamespaceID, RepositoryID: dbRepo.ID}

		// Find corresponding manifest in DB or filesystem.
		var dbManifest *models.Manifest
		dbManifest, err = imp.repositoryStore.FindManifestByDigest(ctx, dbRepo, desc.Digest)
		if err != nil {
			log.WithError(err).Error("finding tag manifest")
			continue
		}
		if dbManifest == nil {
			m, err := manifestService.Get(ctx, desc.Digest)
			if err != nil {
				log.WithError(err).Errorf("retrieving manifest %q", desc.Digest)
				continue
			}

			switch fsManifest := m.(type) {
			case *manifestlist.DeserializedManifestList:
				log.Info("importing manifest list")
				dbManifest, err = imp.importManifestList(ctx, fsRepo, dbRepo, fsManifest, desc.Digest)
			default:
				log.Info("importing manifest")
				dbManifest, err = imp.importManifest(ctx, fsRepo, dbRepo, fsManifest, desc.Digest)
			}
			if err != nil {
				log.WithError(err).Error("importing manifest")
				continue
			}
		}

		dbTag.ManifestID = dbManifest.ID

		if err := imp.tagStore.CreateOrUpdate(ctx, dbTag); err != nil {
			log.WithError(err).Error("creating tag")
		}
	}

	return nil
}

func (imp *Importer) importRepository(ctx context.Context, path string) error {
	named, err := reference.WithName(path)
	if err != nil {
		return fmt.Errorf("parsing repository name: %w", err)
	}
	fsRepo, err := imp.registry.Repository(ctx, named)
	if err != nil {
		return fmt.Errorf("constructing repository: %w", err)
	}

	// Find or create repository.
	var dbRepo *models.Repository

	if dbRepo, err = imp.repositoryStore.CreateOrFindByPath(ctx, path); err != nil {
		return fmt.Errorf("importing repository: %w", err)
	}

	if imp.importDanglingManifests {
		// import all repository manifests
		if err := imp.importManifests(ctx, fsRepo, dbRepo); err != nil {
			return fmt.Errorf("importing manifests: %w", err)
		}
	}

	// import repository tags and associated manifests
	if err := imp.importTags(ctx, fsRepo, dbRepo); err != nil {
		return fmt.Errorf("importing tags: %w", err)
	}

	return nil
}

func (imp *Importer) preImportTaggedManifests(ctx context.Context, fsRepo distribution.Repository, dbRepo *models.Repository) error {
	tagService := fsRepo.Tags(ctx)
	fsTags, err := tagService.All(ctx)
	if err != nil {
		return fmt.Errorf("reading tags: %w", err)
	}

	total := len(fsTags)

	for i, fsTag := range fsTags {
		log := logrus.WithFields(logrus.Fields{"name": fsTag, "count": i + 1, "total": total})

		// read tag details from the filesystem
		desc, err := tagService.Get(ctx, fsTag)
		if err != nil {
			log.WithError(err).Error("reading tag details")
			continue
		}

		// Find corresponding manifest in DB or filesystem.
		var dbManifest *models.Manifest
		dbManifest, err = imp.repositoryStore.FindManifestByDigest(ctx, dbRepo, desc.Digest)
		if err != nil {
			log.WithError(err).Error("finding tag manifest")
			continue
		}
		if dbManifest == nil {
			if err := imp.preImportManifest(ctx, fsRepo, dbRepo, desc.Digest); err != nil {
				return err
			}
		}
	}

	return nil
}

func (imp *Importer) preImportManifest(ctx context.Context, fsRepo distribution.Repository, dbRepo *models.Repository, dgst digest.Digest) error {
	var tx Transactor

	manifestService, err := fsRepo.Manifests(ctx)
	if err != nil {
		return fmt.Errorf("constructing manifest service: %w", err)
	}

	log := logrus.WithField("digest", dgst)

	m, err := manifestService.Get(ctx, dgst)
	if err != nil {
		log.WithError(err).Errorf("retrieving manifest %q", dgst)
	}

	if !imp.dryRun {
		tx, err = imp.beginTx(ctx)
		if err != nil {
			return fmt.Errorf("begin manifest transaction: %w", err)
		}
		defer func() {
			tx.Rollback()
			imp.loadStores(imp.db)
		}()
	}

	switch fsManifest := m.(type) {
	case *manifestlist.DeserializedManifestList:
		log.Info("pre-importing manifest list")
		_, err = imp.importManifestList(ctx, fsRepo, dbRepo, fsManifest, dgst)
	default:
		log.Info("pre-importing manifest")
		_, err = imp.importManifest(ctx, fsRepo, dbRepo, fsManifest, dgst)
	}
	if err != nil {
		log.WithError(err).Error("pre-importing manifest")
	}

	if !imp.dryRun {
		if err := tx.Commit(); err != nil {
			return err
		}
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
	numBlobs, err := imp.blobStore.Count(ctx)
	if err != nil {
		return nil, err
	}
	numTags, err := imp.tagStore.Count(ctx)
	if err != nil {
		return nil, err
	}

	count := map[string]int{
		"repositories": numRepositories,
		"manifests":    numManifests,
		"blobs":        numBlobs,
		"tags":         numTags,
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

// ImportAll populates the registry database with metadata from all repositories in the storage backend.
func (imp *Importer) ImportAll(ctx context.Context) error {
	var tx Transactor
	var err error

	// Create a single transaction and roll it back at the end for dry runs.
	if imp.dryRun {
		tx, err = imp.beginTx(ctx)
		if err != nil {
			return fmt.Errorf("begin dry run transaction: %w", err)
		}
		defer tx.Rollback()
	}

	start := time.Now()
	log := logrus.New()
	log.Info("starting metadata import")

	if imp.requireEmptyDatabase {
		empty, err := imp.isDatabaseEmpty(ctx)
		if err != nil {
			return fmt.Errorf("checking if database is empty: %w", err)
		}
		if !empty {
			return errors.New("non-empty database")
		}
	}

	if imp.importDanglingBlobs {
		var index int
		blobStart := time.Now()
		log.Info("importing all blobs")
		err := imp.registry.Blobs().Enumerate(ctx, func(desc distribution.Descriptor) error {
			index++
			log := log.WithFields(logrus.Fields{"digest": desc.Digest, "count": index, "size": desc.Size})
			log.Info("importing blob")

			dbBlob, err := imp.blobStore.FindByDigest(ctx, desc.Digest)
			if err != nil {
				return fmt.Errorf("checking for existence of blob: %w", err)
			}

			if dbBlob == nil {
				if err := imp.blobStore.Create(ctx, &models.Blob{MediaType: "application/octet-stream", Digest: desc.Digest, Size: desc.Size}); err != nil {
					return err
				}
			}

			// Even if we found the blob in the database, try to transfer in case it's
			// not present in blob storage on the transfer side.
			if err = imp.transferBlob(ctx, desc.Digest); err != nil {
				return fmt.Errorf("transferring blob: %w", err)
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("importing blobs: %w", err)
		}

		blobEnd := time.Since(blobStart).Seconds()
		log.WithField("duration_s", blobEnd).Info("blob import complete")
	}

	repositoryEnumerator, ok := imp.registry.(distribution.RepositoryEnumerator)
	if !ok {
		return errors.New("error building repository enumerator")
	}

	index := 0
	err = repositoryEnumerator.Enumerate(ctx, func(path string) error {
		if !imp.dryRun {
			tx, err = imp.beginTx(ctx)
			if err != nil {
				return fmt.Errorf("begin repository transaction: %w", err)
			}
			defer tx.Rollback()
		}

		index++
		repoStart := time.Now()
		log := logrus.WithFields(logrus.Fields{"path": path, "count": index})
		log.Info("importing repository")

		if err := imp.importRepository(ctx, path); err != nil {
			log.WithError(err).Error("error importing repository")
			// if the storage driver failed to find a repository path (usually due to missing `_manifests/revisions`
			// or `_manifests/tags` folders) continue to the next one, otherwise stop as the error is unknown.
			if !(errors.As(err, &driver.PathNotFoundError{}) || errors.As(err, &distribution.ErrRepositoryUnknown{})) {
				return err
			}
			return nil
		}

		repoEnd := time.Since(repoStart).Seconds()
		log.WithField("duration_s", repoEnd).Info("repository import complete")

		if !imp.dryRun {
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("commit repository transaction: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if !imp.dryRun {
		// reset stores to use the main connection handler instead of the last (committed/rolled back) transaction
		imp.loadStores(imp.db)
	}

	counters, err := imp.countRows(ctx)
	if err != nil {
		logrus.WithError(err).Error("counting table rows")
	}

	logCounters := make(map[string]interface{}, len(counters))
	for t, n := range counters {
		logCounters[t] = n
	}

	t := time.Since(start).Seconds()
	logrus.WithField("duration_s", t).WithFields(logCounters).Info("metadata import complete")

	return err
}

// Import populates the registry database with metadata from a specific repository in the storage backend.
func (imp *Importer) Import(ctx context.Context, path string) error {
	tx, err := imp.beginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin repository transaction: %w", err)
	}
	defer tx.Rollback()

	start := time.Now()
	logrus.Info("starting metadata import")

	if imp.requireEmptyDatabase {
		empty, err := imp.isDatabaseEmpty(ctx)
		if err != nil {
			return fmt.Errorf("checking if database is empty: %w", err)
		}
		if !empty {
			return errors.New("non-empty database")
		}
	}

	log := logrus.WithField("path", path)
	log.Info("importing repository")
	if err := imp.importRepository(ctx, path); err != nil {
		log.WithError(err).Error("error importing repository")
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

	if !imp.dryRun {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit repository transaction: %w", err)
		}
	}

	return err
}

// PreImport populates repository data without including any tag information.
// Running pre-import can reduce the runtime of an Import against the same
// repository and, with online garbage collection enabled, does not require a
// repository to be read-only.
func (imp *Importer) PreImport(ctx context.Context, path string) error {
	var tx Transactor
	var err error

	// Create a single transaction and roll it back at the end for dry runs.
	if imp.dryRun {
		tx, err = imp.beginTx(ctx)
		if err != nil {
			return fmt.Errorf("begin dry run transaction: %w", err)
		}
		defer tx.Rollback()
	}

	if imp.requireEmptyDatabase {
		empty, err := imp.isDatabaseEmpty(ctx)
		if err != nil {
			return fmt.Errorf("checking if database is empty: %w", err)
		}
		if !empty {
			return errors.New("non-empty database")
		}
	}

	start := time.Now()
	log := logrus.WithField("path", path)
	log.Info("starting repository pre-import")

	named, err := reference.WithName(path)
	if err != nil {
		return fmt.Errorf("parsing repository name: %w", err)
	}
	fsRepo, err := imp.registry.Repository(ctx, named)
	if err != nil {
		return fmt.Errorf("constructing repository: %w", err)
	}

	// Find or create repository.
	var dbRepo *models.Repository

	dbRepo, err = imp.repositoryStore.FindByPath(ctx, path)
	if err != nil {
		return fmt.Errorf("checking for existence of repository: %w", err)
	}

	if dbRepo == nil {
		if dbRepo, err = imp.repositoryStore.CreateByPath(ctx, path); err != nil {
			return fmt.Errorf("importing repository: %w", err)
		}
	}

	if err = imp.preImportTaggedManifests(ctx, fsRepo, dbRepo); err != nil {
		return fmt.Errorf("importing tags: %w", err)
	}

	if !imp.dryRun {
		// reset stores to use the main connection handler instead of the last (committed/rolled back) transaction
		imp.loadStores(imp.db)
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
	log.WithField("duration_s", t).WithFields(logCounters).Info("pre-import complete")

	return nil
}
