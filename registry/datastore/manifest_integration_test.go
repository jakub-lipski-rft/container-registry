// +build integration

package datastore_test

import (
	"testing"

	"github.com/docker/distribution/manifest/manifestlist"

	"github.com/docker/distribution/registry/datastore"

	"github.com/docker/distribution/registry/datastore/models"
	"github.com/docker/distribution/registry/datastore/testutil"
	"github.com/stretchr/testify/require"
)

func reloadManifestFixtures(tb testing.TB) {
	testutil.ReloadFixtures(
		tb, suite.db, suite.basePath,
		// Manifest has a relationship with Repository and ManifestLayer (insert order matters)
		testutil.RepositoriesTable, testutil.BlobsTable, testutil.ManifestsTable,
		testutil.TagsTable, testutil.ManifestReferencesTable, testutil.LayersTable,
	)
}

func unloadManifestFixtures(tb testing.TB) {
	require.NoError(tb, testutil.TruncateTables(
		suite.db,
		// Manifest has a relationship with Repository and ManifestLayer (insert order matters)
		testutil.RepositoriesTable, testutil.BlobsTable, testutil.ManifestsTable,
		testutil.TagsTable, testutil.ManifestReferencesTable, testutil.LayersTable,
	))
}

func TestManifestStore_ImplementsReaderAndWriter(t *testing.T) {
	require.Implements(t, (*datastore.ManifestStore)(nil), datastore.NewManifestStore(suite.db))
}

func TestManifestStore_FindByID(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)

	m, err := s.FindByID(suite.ctx, 1)
	require.NoError(t, err)

	// see testdata/fixtures/manifests.sql
	expected := &models.Manifest{
		ID:            1,
		RepositoryID:  3,
		SchemaVersion: 2,
		MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
		Digest:        "sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155",
		Payload:       models.Payload(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1640,"digest":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"}]}`),
		Configuration: &models.Configuration{
			MediaType: "application/vnd.docker.container.image.v1+json",
			Digest:    "sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9",
			Payload:   models.Payload(`{"architecture":"amd64","config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh"],"ArgsEscaped":true,"Image":"sha256:e7d92cdc71feacf90708cb59182d0df1b911f8ae022d29e8e95d75ca6a99776a","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"container":"7980908783eb05384926afb5ffad45856f65bc30029722a4be9f1eb3661e9c5e","container_config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh","-c","echo \"1\" \u003e /data"],"Image":"sha256:e7d92cdc71feacf90708cb59182d0df1b911f8ae022d29e8e95d75ca6a99776a","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"created":"2020-03-02T12:21:53.8027967Z","docker_version":"19.03.5","history":[{"created":"2020-01-18T01:19:37.02673981Z","created_by":"/bin/sh -c #(nop) ADD file:e69d441d729412d24675dcd33e04580885df99981cec43de8c9b24015313ff8e in / "},{"created":"2020-01-18T01:19:37.187497623Z","created_by":"/bin/sh -c #(nop)  CMD [\"/bin/sh\"]","empty_layer":true},{"created":"2020-03-02T12:21:53.8027967Z","created_by":"/bin/sh -c echo \"1\" \u003e /data"}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:5216338b40a7b96416b8b9858974bbe4acc3096ee60acbc4dfb1ee02aecceb10","sha256:99cb4c5d9f96432a00201f4b14c058c6235e563917ba7af8ed6c4775afa5780f"]}}`),
		},
		CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", m.CreatedAt.Location()),
	}
	require.Equal(t, expected, m)
}

func TestManifestStore_FindByID_NotFound(t *testing.T) {
	s := datastore.NewManifestStore(suite.db)

	m, err := s.FindByID(suite.ctx, 0)
	require.Nil(t, m)
	require.NoError(t, err)
}

func TestManifestStore_FindByDigest(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)

	m, err := s.FindByDigest(suite.ctx, "sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f")
	require.NoError(t, err)

	// see testdata/fixtures/manifests.sql
	excepted := &models.Manifest{
		ID:            2,
		RepositoryID:  3,
		SchemaVersion: 2,
		MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
		Digest:        "sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f",
		Payload:       models.Payload(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1819,"digest":"sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":109,"digest":"sha256:f01256086224ded321e042e74135d72d5f108089a1cda03ab4820dfc442807c1"}]}`),
		Configuration: &models.Configuration{
			MediaType: "application/vnd.docker.container.image.v1+json",
			Digest:    "sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073",
			Payload:   models.Payload(`{"architecture":"amd64","config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh"],"ArgsEscaped":true,"Image":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"container":"cb78c8a8058712726096a7a8f80e6a868ffb514a07f4fef37639f42d99d997e4","container_config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh","-c","echo \"2\" \u003e\u003e /data"],"Image":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"created":"2020-03-02T12:24:16.7039823Z","docker_version":"19.03.5","history":[{"created":"2020-01-18T01:19:37.02673981Z","created_by":"/bin/sh -c #(nop) ADD file:e69d441d729412d24675dcd33e04580885df99981cec43de8c9b24015313ff8e in / "},{"created":"2020-01-18T01:19:37.187497623Z","created_by":"/bin/sh -c #(nop)  CMD [\"/bin/sh\"]","empty_layer":true},{"created":"2020-03-02T12:21:53.8027967Z","created_by":"/bin/sh -c echo \"1\" \u003e /data"},{"created":"2020-03-02T12:24:16.7039823Z","created_by":"/bin/sh -c echo \"2\" \u003e\u003e /data"}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:5216338b40a7b96416b8b9858974bbe4acc3096ee60acbc4dfb1ee02aecceb10","sha256:99cb4c5d9f96432a00201f4b14c058c6235e563917ba7af8ed6c4775afa5780f","sha256:6322c07f5c6ad456f64647993dfc44526f4548685ee0f3d8f03534272b3a06d8"]}}`),
		},
		CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", m.CreatedAt.Location()),
	}
	require.Equal(t, excepted, m)
}

func TestManifestStore_FindByDigest_NotFound(t *testing.T) {
	s := datastore.NewManifestStore(suite.db)

	m, err := s.FindByDigest(suite.ctx, "sha256:16b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f")
	require.Nil(t, m)
	require.NoError(t, err)
}

func TestManifestStore_FindAll(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)

	mm, err := s.FindAll(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/manifests.sql
	local := mm[0].CreatedAt.Location()
	expected := models.Manifests{
		{
			ID:            1,
			RepositoryID:  3,
			SchemaVersion: 2,
			MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
			Digest:        "sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155",
			Payload:       models.Payload(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1640,"digest":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"}]}`),
			Configuration: &models.Configuration{
				MediaType: "application/vnd.docker.container.image.v1+json",
				Digest:    "sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9",
				Payload:   models.Payload(`{"architecture":"amd64","config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh"],"ArgsEscaped":true,"Image":"sha256:e7d92cdc71feacf90708cb59182d0df1b911f8ae022d29e8e95d75ca6a99776a","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"container":"7980908783eb05384926afb5ffad45856f65bc30029722a4be9f1eb3661e9c5e","container_config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh","-c","echo \"1\" \u003e /data"],"Image":"sha256:e7d92cdc71feacf90708cb59182d0df1b911f8ae022d29e8e95d75ca6a99776a","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"created":"2020-03-02T12:21:53.8027967Z","docker_version":"19.03.5","history":[{"created":"2020-01-18T01:19:37.02673981Z","created_by":"/bin/sh -c #(nop) ADD file:e69d441d729412d24675dcd33e04580885df99981cec43de8c9b24015313ff8e in / "},{"created":"2020-01-18T01:19:37.187497623Z","created_by":"/bin/sh -c #(nop)  CMD [\"/bin/sh\"]","empty_layer":true},{"created":"2020-03-02T12:21:53.8027967Z","created_by":"/bin/sh -c echo \"1\" \u003e /data"}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:5216338b40a7b96416b8b9858974bbe4acc3096ee60acbc4dfb1ee02aecceb10","sha256:99cb4c5d9f96432a00201f4b14c058c6235e563917ba7af8ed6c4775afa5780f"]}}`),
			},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", local),
		},
		{
			ID:            2,
			RepositoryID:  3,
			SchemaVersion: 2,
			MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
			Digest:        "sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f",
			Payload:       models.Payload(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1819,"digest":"sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":109,"digest":"sha256:f01256086224ded321e042e74135d72d5f108089a1cda03ab4820dfc442807c1"}]}`),
			Configuration: &models.Configuration{
				MediaType: "application/vnd.docker.container.image.v1+json",
				Digest:    "sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073",
				Payload:   models.Payload(`{"architecture":"amd64","config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh"],"ArgsEscaped":true,"Image":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"container":"cb78c8a8058712726096a7a8f80e6a868ffb514a07f4fef37639f42d99d997e4","container_config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh","-c","echo \"2\" \u003e\u003e /data"],"Image":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"created":"2020-03-02T12:24:16.7039823Z","docker_version":"19.03.5","history":[{"created":"2020-01-18T01:19:37.02673981Z","created_by":"/bin/sh -c #(nop) ADD file:e69d441d729412d24675dcd33e04580885df99981cec43de8c9b24015313ff8e in / "},{"created":"2020-01-18T01:19:37.187497623Z","created_by":"/bin/sh -c #(nop)  CMD [\"/bin/sh\"]","empty_layer":true},{"created":"2020-03-02T12:21:53.8027967Z","created_by":"/bin/sh -c echo \"1\" \u003e /data"},{"created":"2020-03-02T12:24:16.7039823Z","created_by":"/bin/sh -c echo \"2\" \u003e\u003e /data"}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:5216338b40a7b96416b8b9858974bbe4acc3096ee60acbc4dfb1ee02aecceb10","sha256:99cb4c5d9f96432a00201f4b14c058c6235e563917ba7af8ed6c4775afa5780f","sha256:6322c07f5c6ad456f64647993dfc44526f4548685ee0f3d8f03534272b3a06d8"]}}`),
			},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", local),
		},
		{
			ID:            3,
			RepositoryID:  4,
			SchemaVersion: 2,
			MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
			Digest:        "sha256:bca3c0bf2ca0cde987ad9cab2dac986047a0ccff282f1b23df282ef05e3a10a6",
			Payload:       models.Payload(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":6775,"digest":"sha256:33f3ef3322b28ecfc368872e621ab715a04865471c47ca7426f3e93846157780"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":27091819,"digest":"sha256:68ced04f60ab5c7a5f1d0b0b4e7572c5a4c8cce44866513d30d9df1a15277d6b"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":23882259,"digest":"sha256:c4039fd85dccc8e267c98447f8f1b27a402dbb4259d86586f4097acb5e6634af"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":203,"digest":"sha256:c16ce02d3d6132f7059bf7e9ff6205cbf43e86c538ef981c37598afd27d01efa"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":107,"digest":"sha256:a0696058fc76fe6f456289f5611efe5c3411814e686f59f28b2e2069ed9e7d28"}]}`),
			Configuration: &models.Configuration{
				MediaType: "application/vnd.docker.container.image.v1+json",
				Digest:    "sha256:33f3ef3322b28ecfc368872e621ab715a04865471c47ca7426f3e93846157780",
				Payload:   models.Payload(`{"architecture":"amd64","config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"ExposedPorts":{"80/tcp":{}},"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin","NGINX_VERSION=1.17.8","NJS_VERSION=0.3.8","PKG_RELEASE=1~buster"],"Cmd":["nginx","-g","daemon off;"],"ArgsEscaped":true,"Image":"sha256:a1523e859360df9ffe2b31a8270f5e16422609fe138c1636383efdc34b9ea2d6","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":{"maintainer":"NGINX Docker Maintainers \u003cdocker-maint@nginx.com\u003e"},"StopSignal":"SIGTERM"},"container":"9a24d3f0d5ca79fceaef1956a91e0ba05b2e924295b8b0ec439db5a6bd491dda","container_config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"ExposedPorts":{"80/tcp":{}},"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin","NGINX_VERSION=1.17.8","NJS_VERSION=0.3.8","PKG_RELEASE=1~buster"],"Cmd":["/bin/sh","-c","echo \"1\" \u003e /data"],"Image":"sha256:a1523e859360df9ffe2b31a8270f5e16422609fe138c1636383efdc34b9ea2d6","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":{"maintainer":"NGINX Docker Maintainers \u003cdocker-maint@nginx.com\u003e"},"StopSignal":"SIGTERM"},"created":"2020-03-02T12:34:49.9572024Z","docker_version":"19.03.5","history":[{"created":"2020-02-26T00:37:39.301941924Z","created_by":"/bin/sh -c #(nop) ADD file:e5a364615e0f6961626089c7d658adbf8c8d95b3ae95a390a8bb33875317d434 in / "},{"created":"2020-02-26T00:37:39.539684396Z","created_by":"/bin/sh -c #(nop)  CMD [\"bash\"]","empty_layer":true},{"created":"2020-02-26T20:01:52.907016299Z","created_by":"/bin/sh -c #(nop)  LABEL maintainer=NGINX Docker Maintainers \u003cdocker-maint@nginx.com\u003e","empty_layer":true},{"created":"2020-02-26T20:01:53.114563769Z","created_by":"/bin/sh -c #(nop)  ENV NGINX_VERSION=1.17.8","empty_layer":true},{"created":"2020-02-26T20:01:53.28669526Z","created_by":"/bin/sh -c #(nop)  ENV NJS_VERSION=0.3.8","empty_layer":true},{"created":"2020-02-26T20:01:53.470888291Z","created_by":"/bin/sh -c #(nop)  ENV PKG_RELEASE=1~buster","empty_layer":true},{"created":"2020-02-26T20:02:14.311730686Z","created_by":"/bin/sh -c set -x     \u0026\u0026 addgroup --system --gid 101 nginx     \u0026\u0026 adduser --system --disabled-login --ingroup nginx --no-create-home --home /nonexistent --gecos \"nginx user\" --shell /bin/false --uid 101 nginx     \u0026\u0026 apt-get update     \u0026\u0026 apt-get install --no-install-recommends --no-install-suggests -y gnupg1 ca-certificates     \u0026\u0026     NGINX_GPGKEY=573BFD6B3D8FBC641079A6ABABF5BD827BD9BF62;     found='';     for server in         ha.pool.sks-keyservers.net         hkp://keyserver.ubuntu.com:80         hkp://p80.pool.sks-keyservers.net:80         pgp.mit.edu     ; do         echo \"Fetching GPG key $NGINX_GPGKEY from $server\";         apt-key adv --keyserver \"$server\" --keyserver-options timeout=10 --recv-keys \"$NGINX_GPGKEY\" \u0026\u0026 found=yes \u0026\u0026 break;     done;     test -z \"$found\" \u0026\u0026 echo \u003e\u00262 \"error: failed to fetch GPG key $NGINX_GPGKEY\" \u0026\u0026 exit 1;     apt-get remove --purge --auto-remove -y gnupg1 \u0026\u0026 rm -rf /var/lib/apt/lists/*     \u0026\u0026 dpkgArch=\"$(dpkg --print-architecture)\"     \u0026\u0026 nginxPackages=\"         nginx=${NGINX_VERSION}-${PKG_RELEASE}         nginx-module-xslt=${NGINX_VERSION}-${PKG_RELEASE}         nginx-module-geoip=${NGINX_VERSION}-${PKG_RELEASE}         nginx-module-image-filter=${NGINX_VERSION}-${PKG_RELEASE}         nginx-module-njs=${NGINX_VERSION}.${NJS_VERSION}-${PKG_RELEASE}     \"     \u0026\u0026 case \"$dpkgArch\" in         amd64|i386)             echo \"deb https://nginx.org/packages/mainline/debian/ buster nginx\" \u003e\u003e /etc/apt/sources.list.d/nginx.list             \u0026\u0026 apt-get update             ;;         *)             echo \"deb-src https://nginx.org/packages/mainline/debian/ buster nginx\" \u003e\u003e /etc/apt/sources.list.d/nginx.list                         \u0026\u0026 tempDir=\"$(mktemp -d)\"             \u0026\u0026 chmod 777 \"$tempDir\"                         \u0026\u0026 savedAptMark=\"$(apt-mark showmanual)\"                         \u0026\u0026 apt-get update             \u0026\u0026 apt-get build-dep -y $nginxPackages             \u0026\u0026 (                 cd \"$tempDir\"                 \u0026\u0026 DEB_BUILD_OPTIONS=\"nocheck parallel=$(nproc)\"                     apt-get source --compile $nginxPackages             )                         \u0026\u0026 apt-mark showmanual | xargs apt-mark auto \u003e /dev/null             \u0026\u0026 { [ -z \"$savedAptMark\" ] || apt-mark manual $savedAptMark; }                         \u0026\u0026 ls -lAFh \"$tempDir\"             \u0026\u0026 ( cd \"$tempDir\" \u0026\u0026 dpkg-scanpackages . \u003e Packages )             \u0026\u0026 grep '^Package: ' \"$tempDir/Packages\"             \u0026\u0026 echo \"deb [ trusted=yes ] file://$tempDir ./\" \u003e /etc/apt/sources.list.d/temp.list             \u0026\u0026 apt-get -o Acquire::GzipIndexes=false update             ;;     esac         \u0026\u0026 apt-get install --no-install-recommends --no-install-suggests -y                         $nginxPackages                         gettext-base     \u0026\u0026 apt-get remove --purge --auto-remove -y ca-certificates \u0026\u0026 rm -rf /var/lib/apt/lists/* /etc/apt/sources.list.d/nginx.list         \u0026\u0026 if [ -n \"$tempDir\" ]; then         apt-get purge -y --auto-remove         \u0026\u0026 rm -rf \"$tempDir\" /etc/apt/sources.list.d/temp.list;     fi"},{"created":"2020-02-26T20:02:15.146823517Z","created_by":"/bin/sh -c ln -sf /dev/stdout /var/log/nginx/access.log     \u0026\u0026 ln -sf /dev/stderr /var/log/nginx/error.log"},{"created":"2020-02-26T20:02:15.335986561Z","created_by":"/bin/sh -c #(nop)  EXPOSE 80","empty_layer":true},{"created":"2020-02-26T20:02:15.543209017Z","created_by":"/bin/sh -c #(nop)  STOPSIGNAL SIGTERM","empty_layer":true},{"created":"2020-02-26T20:02:15.724396212Z","created_by":"/bin/sh -c #(nop)  CMD [\"nginx\" \"-g\" \"daemon off;\"]","empty_layer":true},{"created":"2020-03-02T12:34:49.9572024Z","created_by":"/bin/sh -c echo \"1\" \u003e /data"}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:f2cb0ecef392f2a630fa1205b874ab2e2aedf96de04d0b8838e4e728e28142da","sha256:fe08d5d042ab93bee05f9cda17f1c57066e146b0704be2ff755d14c25e6aa5e8","sha256:318be7aea8fc62d5910cca0d49311fa8d95502c90e2a91b7a4d78032a670b644","sha256:ca5cd87c6bf8376275e0bf32cd7139ed17dd69ef28bda9ba15d07475b147f931"]}}`),
			},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", local),
		},
		{
			ID:            4,
			RepositoryID:  4,
			SchemaVersion: 1,
			MediaType:     "application/vnd.docker.distribution.manifest.v1+json",
			Digest:        "sha256:ea1650093606d9e76dfc78b986d57daea6108af2d5a9114a98d7198548bfdfc7",
			Payload:       models.Payload(`{"schemaVersion":1,"name":"gitlab-org/gitlab-test/frontend","tag":"0.0.1","architecture":"amd64","fsLayers":[{"blobSum":"sha256:68ced04f60ab5c7a5f1d0b0b4e7572c5a4c8cce44866513d30d9df1a15277d6b"},{"blobSum":"sha256:c4039fd85dccc8e267c98447f8f1b27a402dbb4259d86586f4097acb5e6634af"},{"blobSum":"sha256:c16ce02d3d6132f7059bf7e9ff6205cbf43e86c538ef981c37598afd27d01efa"}],"history":[{"v1Compatibility":"{\"architecture\":\"amd64\",\"config\":{\"Hostname\":\"\",\"Domainname\":\"\",\"User\":\"\",\"AttachStdin\":false,\"AttachStdout\":false,\"AttachStderr\":false,\"Tty\":false,\"OpenStdin\":false,\"StdinOnce\":false,\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"],\"Cmd\":[\"/bin/sh\"],\"ArgsEscaped\":true,\"Image\":\"sha256:74df73bb19fbfc7fb5ab9a8234b3d98ee2fb92df5b824496679802685205ab8c\",\"Volumes\":null,\"WorkingDir\":\"\",\"Entrypoint\":null,\"OnBuild\":null,\"Labels\":null},\"container\":\"fb71ddde5f6411a82eb056a9190f0cc1c80d7f77a8509ee90a2054428edb0024\",\"container_config\":{\"Hostname\":\"fb71ddde5f64\",\"Domainname\":\"\",\"User\":\"\",\"AttachStdin\":false,\"AttachStdout\":false,\"AttachStderr\":false,\"Tty\":false,\"OpenStdin\":false,\"StdinOnce\":false,\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"],\"Cmd\":[\"/bin/sh\",\"-c\",\"#(nop) \",\"CMD [\\\"/bin/sh\\\"]\"],\"ArgsEscaped\":true,\"Image\":\"sha256:74df73bb19fbfc7fb5ab9a8234b3d98ee2fb92df5b824496679802685205ab8c\",\"Volumes\":null,\"WorkingDir\":\"\",\"Entrypoint\":null,\"OnBuild\":null,\"Labels\":{}},\"created\":\"2020-03-23T21:19:34.196162891Z\",\"docker_version\":\"18.09.7\",\"id\":\"13787be01505ffa9179a780b616b953330baedfca1667797057aa3af67e8b39d\",\"os\":\"linux\",\"parent\":\"c6875a916c6940e6590b05b29f484059b82e19ca0eed100e2e805aebd98614b8\",\"throwaway\":true}"},{"v1Compatibility":"{\"id\":\"c6875a916c6940e6590b05b29f484059b82e19ca0eed100e2e805aebd98614b8\",\"created\":\"2020-03-23T21:19:34.027725872Z\",\"container_config\":{\"Cmd\":[\"/bin/sh -c #(nop) ADD file:0c4555f363c2672e350001f1293e689875a3760afe7b3f9146886afe67121cba in / \"]}}"}],"signatures":[{"header":{"jwk":{"crv":"P-256","kid":"SVNG:A2VR:TQJG:H626:HBKH:6WBU:GFBH:3YNI:425G:MDXK:ULXZ:CENN","kty":"EC","x":"daLesX_y73FSCFCaBuCR8offV_m7XEohHZJ9z-6WvOM","y":"pLEDUlQMDiEQqheWYVC55BPIB0m8BIhI-fxQBCH_wA0"},"alg":"ES256"},"signature":"mqA4qF-St1HTNsjHzhgnHBeN38ptKJOi4wSeH4xc_FCEPv0OchAUJC6v2gYTP4TwostmX-AB1_z3jo9G_ZuX5w","protected":"eyJmb3JtYXRMZW5ndGgiOjIxNTQsImZvcm1hdFRhaWwiOiJDbjAiLCJ0aW1lIjoiMjAyMC0wNC0xNVQwODoxMzowNVoifQ"}]}`),
			CreatedAt:     testutil.ParseTimestamp(t, "2020-04-15 09:47:26.461413", local),
		},
		{
			ID:            5,
			RepositoryID:  6,
			SchemaVersion: 2,
			MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
			Digest:        "sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155",
			Payload:       models.Payload(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1640,"digest":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"}]}`),
			Configuration: &models.Configuration{
				MediaType: "application/vnd.docker.container.image.v1+json",
				Digest:    "sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9",
				Payload:   models.Payload(`{"architecture":"amd64","config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh"],"ArgsEscaped":true,"Image":"sha256:e7d92cdc71feacf90708cb59182d0df1b911f8ae022d29e8e95d75ca6a99776a","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"container":"7980908783eb05384926afb5ffad45856f65bc30029722a4be9f1eb3661e9c5e","container_config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh","-c","echo \"1\" \u003e /data"],"Image":"sha256:e7d92cdc71feacf90708cb59182d0df1b911f8ae022d29e8e95d75ca6a99776a","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"created":"2020-03-02T12:21:53.8027967Z","docker_version":"19.03.5","history":[{"created":"2020-01-18T01:19:37.02673981Z","created_by":"/bin/sh -c #(nop) ADD file:e69d441d729412d24675dcd33e04580885df99981cec43de8c9b24015313ff8e in / "},{"created":"2020-01-18T01:19:37.187497623Z","created_by":"/bin/sh -c #(nop)  CMD [\"/bin/sh\"]","empty_layer":true},{"created":"2020-03-02T12:21:53.8027967Z","created_by":"/bin/sh -c echo \"1\" \u003e /data"}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:5216338b40a7b96416b8b9858974bbe4acc3096ee60acbc4dfb1ee02aecceb10","sha256:99cb4c5d9f96432a00201f4b14c058c6235e563917ba7af8ed6c4775afa5780f"]}}`),
			},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", local),
		},
		{
			ID:            6,
			RepositoryID:  3,
			SchemaVersion: 2,
			MediaType:     manifestlist.MediaTypeManifestList,
			Digest:        "sha256:dc27c897a7e24710a2821878456d56f3965df7cc27398460aa6f21f8b385d2d0",
			Payload:       models.Payload(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.list.v2+json","manifests":[{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":23321,"digest":"sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155","platform":{"architecture":"amd64","os":"linux"}},{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":24123,"digest":"sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f","platform":{"architecture":"amd64","os":"windows","os.version":"10.0.14393.2189"}}]}`),
			CreatedAt:     testutil.ParseTimestamp(t, "2020-04-02 18:45:03.470711", local),
		},
		{
			ID:            7,
			RepositoryID:  4,
			SchemaVersion: 2,
			MediaType:     manifestlist.MediaTypeManifestList,
			Digest:        "sha256:45e85a20d32f249c323ed4085026b6b0ee264788276aa7c06cf4b5da1669067a",
			Payload:       models.Payload(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.list.v2+json","manifests":[{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":24123,"digest":"sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f","platform":{"architecture":"amd64","os":"windows","os.version":"10.0.14393.2189"}},{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":42212,"digest":"sha256:bca3c0bf2ca0cde987ad9cab2dac986047a0ccff282f1b23df282ef05e3a10a6","platform":{"architecture":"amd64","os":"linux"}}]}`),
			CreatedAt:     testutil.ParseTimestamp(t, "2020-04-02 18:45:04.470711", local),
		},
		{
			ID:            8,
			RepositoryID:  7,
			SchemaVersion: 2,
			MediaType:     manifestlist.MediaTypeManifestList,
			Digest:        "sha256:dc27c897a7e24710a2821878456d56f3965df7cc27398460aa6f21f8b385d2d0",
			Payload:       models.Payload(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.list.v2+json","manifests":[{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":23321,"digest":"sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155","platform":{"architecture":"amd64","os":"linux"}},{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":24123,"digest":"sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f","platform":{"architecture":"amd64","os":"windows","os.version":"10.0.14393.2189"}}]}`),
			CreatedAt:     testutil.ParseTimestamp(t, "2020-04-02 18:45:03.470711", local),
		},
		{
			ID:            9,
			RepositoryID:  4,
			SchemaVersion: 2,
			MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
			Digest:        "sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f",
			Payload:       models.Payload(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1819,"digest":"sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":109,"digest":"sha256:f01256086224ded321e042e74135d72d5f108089a1cda03ab4820dfc442807c1"}]}`),
			Configuration: &models.Configuration{
				MediaType: "application/vnd.docker.container.image.v1+json",
				Digest:    "sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073",
				Payload:   models.Payload(`{"architecture":"amd64","config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh"],"ArgsEscaped":true,"Image":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"container":"cb78c8a8058712726096a7a8f80e6a868ffb514a07f4fef37639f42d99d997e4","container_config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh","-c","echo \"2\" \u003e\u003e /data"],"Image":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"created":"2020-03-02T12:24:16.7039823Z","docker_version":"19.03.5","history":[{"created":"2020-01-18T01:19:37.02673981Z","created_by":"/bin/sh -c #(nop) ADD file:e69d441d729412d24675dcd33e04580885df99981cec43de8c9b24015313ff8e in / "},{"created":"2020-01-18T01:19:37.187497623Z","created_by":"/bin/sh -c #(nop)  CMD [\"/bin/sh\"]","empty_layer":true},{"created":"2020-03-02T12:21:53.8027967Z","created_by":"/bin/sh -c echo \"1\" \u003e /data"},{"created":"2020-03-02T12:24:16.7039823Z","created_by":"/bin/sh -c echo \"2\" \u003e\u003e /data"}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:5216338b40a7b96416b8b9858974bbe4acc3096ee60acbc4dfb1ee02aecceb10","sha256:99cb4c5d9f96432a00201f4b14c058c6235e563917ba7af8ed6c4775afa5780f","sha256:6322c07f5c6ad456f64647993dfc44526f4548685ee0f3d8f03534272b3a06d8"]}}`),
			},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", local),
		},
		{
			ID:            10,
			RepositoryID:  7,
			SchemaVersion: 2,
			MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
			Digest:        "sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155",
			Payload:       models.Payload(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1640,"digest":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"}]}`),
			Configuration: &models.Configuration{
				MediaType: "application/vnd.docker.container.image.v1+json",
				Digest:    "sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9",
				Payload:   models.Payload(`{"architecture":"amd64","config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh"],"ArgsEscaped":true,"Image":"sha256:e7d92cdc71feacf90708cb59182d0df1b911f8ae022d29e8e95d75ca6a99776a","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"container":"7980908783eb05384926afb5ffad45856f65bc30029722a4be9f1eb3661e9c5e","container_config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh","-c","echo \"1\" \u003e /data"],"Image":"sha256:e7d92cdc71feacf90708cb59182d0df1b911f8ae022d29e8e95d75ca6a99776a","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"created":"2020-03-02T12:21:53.8027967Z","docker_version":"19.03.5","history":[{"created":"2020-01-18T01:19:37.02673981Z","created_by":"/bin/sh -c #(nop) ADD file:e69d441d729412d24675dcd33e04580885df99981cec43de8c9b24015313ff8e in / "},{"created":"2020-01-18T01:19:37.187497623Z","created_by":"/bin/sh -c #(nop)  CMD [\"/bin/sh\"]","empty_layer":true},{"created":"2020-03-02T12:21:53.8027967Z","created_by":"/bin/sh -c echo \"1\" \u003e /data"}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:5216338b40a7b96416b8b9858974bbe4acc3096ee60acbc4dfb1ee02aecceb10","sha256:99cb4c5d9f96432a00201f4b14c058c6235e563917ba7af8ed6c4775afa5780f"]}}`),
			},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", local),
		},
		{
			ID:            11,
			RepositoryID:  7,
			SchemaVersion: 2,
			MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
			Digest:        "sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f",
			Payload:       models.Payload(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1819,"digest":"sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":109,"digest":"sha256:f01256086224ded321e042e74135d72d5f108089a1cda03ab4820dfc442807c1"}]}`),
			Configuration: &models.Configuration{
				MediaType: "application/vnd.docker.container.image.v1+json",
				Digest:    "sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073",
				Payload:   models.Payload(`{"architecture":"amd64","config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh"],"ArgsEscaped":true,"Image":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"container":"cb78c8a8058712726096a7a8f80e6a868ffb514a07f4fef37639f42d99d997e4","container_config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh","-c","echo \"2\" \u003e\u003e /data"],"Image":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"created":"2020-03-02T12:24:16.7039823Z","docker_version":"19.03.5","history":[{"created":"2020-01-18T01:19:37.02673981Z","created_by":"/bin/sh -c #(nop) ADD file:e69d441d729412d24675dcd33e04580885df99981cec43de8c9b24015313ff8e in / "},{"created":"2020-01-18T01:19:37.187497623Z","created_by":"/bin/sh -c #(nop)  CMD [\"/bin/sh\"]","empty_layer":true},{"created":"2020-03-02T12:21:53.8027967Z","created_by":"/bin/sh -c echo \"1\" \u003e /data"},{"created":"2020-03-02T12:24:16.7039823Z","created_by":"/bin/sh -c echo \"2\" \u003e\u003e /data"}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:5216338b40a7b96416b8b9858974bbe4acc3096ee60acbc4dfb1ee02aecceb10","sha256:99cb4c5d9f96432a00201f4b14c058c6235e563917ba7af8ed6c4775afa5780f","sha256:6322c07f5c6ad456f64647993dfc44526f4548685ee0f3d8f03534272b3a06d8"]}}`),
			},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", local),
		},
	}
	require.Equal(t, expected, mm)
}

func TestManifestStore_FindAll_NotFound(t *testing.T) {
	unloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)

	mm, err := s.FindAll(suite.ctx)
	require.Empty(t, mm)
	require.NoError(t, err)
}

func TestManifestStore_Count(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)
	count, err := s.Count(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/manifests.sql
	require.Equal(t, 11, count)
}

func TestManifestStore_Layers(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)
	bb, err := s.LayerBlobs(suite.ctx, &models.Manifest{ID: 1})
	require.NoError(t, err)

	// see testdata/fixtures/layers.sql
	local := bb[0].CreatedAt.Location()
	expected := models.Blobs{
		{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9",
			Size:      2802957,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:05:35.338639", local),
		},
		{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21",
			Size:      108,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:05:35.338639", local),
		},
	}
	require.Equal(t, expected, bb)
}

func TestManifestStore_References(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)

	// see testdata/fixtures/manifest_references.sql
	ml := &models.Manifest{RepositoryID: 3, ID: 6}
	mm, err := s.References(suite.ctx, ml)
	require.NoError(t, err)

	local := mm[0].CreatedAt.Location()
	expected := models.Manifests{
		{
			ID:            1,
			RepositoryID:  3,
			SchemaVersion: 2,
			MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
			Digest:        "sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155",
			Payload:       models.Payload(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1640,"digest":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"}]}`),
			Configuration: &models.Configuration{
				MediaType: "application/vnd.docker.container.image.v1+json",
				Digest:    "sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9",
				Payload:   models.Payload(`{"architecture":"amd64","config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh"],"ArgsEscaped":true,"Image":"sha256:e7d92cdc71feacf90708cb59182d0df1b911f8ae022d29e8e95d75ca6a99776a","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"container":"7980908783eb05384926afb5ffad45856f65bc30029722a4be9f1eb3661e9c5e","container_config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh","-c","echo \"1\" \u003e /data"],"Image":"sha256:e7d92cdc71feacf90708cb59182d0df1b911f8ae022d29e8e95d75ca6a99776a","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"created":"2020-03-02T12:21:53.8027967Z","docker_version":"19.03.5","history":[{"created":"2020-01-18T01:19:37.02673981Z","created_by":"/bin/sh -c #(nop) ADD file:e69d441d729412d24675dcd33e04580885df99981cec43de8c9b24015313ff8e in / "},{"created":"2020-01-18T01:19:37.187497623Z","created_by":"/bin/sh -c #(nop)  CMD [\"/bin/sh\"]","empty_layer":true},{"created":"2020-03-02T12:21:53.8027967Z","created_by":"/bin/sh -c echo \"1\" \u003e /data"}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:5216338b40a7b96416b8b9858974bbe4acc3096ee60acbc4dfb1ee02aecceb10","sha256:99cb4c5d9f96432a00201f4b14c058c6235e563917ba7af8ed6c4775afa5780f"]}}`),
			},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", local),
		},
		{
			ID:            2,
			RepositoryID:  3,
			SchemaVersion: 2,
			MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
			Digest:        "sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f",
			Payload:       models.Payload(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1819,"digest":"sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":109,"digest":"sha256:f01256086224ded321e042e74135d72d5f108089a1cda03ab4820dfc442807c1"}]}`),
			Configuration: &models.Configuration{
				MediaType: "application/vnd.docker.container.image.v1+json",
				Digest:    "sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073",
				Payload:   models.Payload(`{"architecture":"amd64","config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh"],"ArgsEscaped":true,"Image":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"container":"cb78c8a8058712726096a7a8f80e6a868ffb514a07f4fef37639f42d99d997e4","container_config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh","-c","echo \"2\" \u003e\u003e /data"],"Image":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"created":"2020-03-02T12:24:16.7039823Z","docker_version":"19.03.5","history":[{"created":"2020-01-18T01:19:37.02673981Z","created_by":"/bin/sh -c #(nop) ADD file:e69d441d729412d24675dcd33e04580885df99981cec43de8c9b24015313ff8e in / "},{"created":"2020-01-18T01:19:37.187497623Z","created_by":"/bin/sh -c #(nop)  CMD [\"/bin/sh\"]","empty_layer":true},{"created":"2020-03-02T12:21:53.8027967Z","created_by":"/bin/sh -c echo \"1\" \u003e /data"},{"created":"2020-03-02T12:24:16.7039823Z","created_by":"/bin/sh -c echo \"2\" \u003e\u003e /data"}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:5216338b40a7b96416b8b9858974bbe4acc3096ee60acbc4dfb1ee02aecceb10","sha256:99cb4c5d9f96432a00201f4b14c058c6235e563917ba7af8ed6c4775afa5780f","sha256:6322c07f5c6ad456f64647993dfc44526f4548685ee0f3d8f03534272b3a06d8"]}}`),
			},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", local),
		},
	}
	require.Equal(t, expected, mm)
}

func TestManifestStore_References_None(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)

	// see testdata/fixtures/manifest_references.sql
	ml := &models.Manifest{ID: 5}
	err := s.DissociateManifest(suite.ctx, ml, &models.Manifest{ID: 1})
	require.NoError(t, err)
	err = s.DissociateManifest(suite.ctx, ml, &models.Manifest{ID: 2})
	require.NoError(t, err)

	mm, err := s.References(suite.ctx, ml)
	require.NoError(t, err)
	require.Empty(t, mm)
}

func TestManifestStore_Create(t *testing.T) {
	reloadManifestFixtures(t)
	require.NoError(t, testutil.TruncateTables(suite.db, testutil.ManifestsTable))

	s := datastore.NewManifestStore(suite.db)
	m := &models.Manifest{
		RepositoryID:  7,
		SchemaVersion: 2,
		MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
		Digest:        "sha256:46b163863b462eadc1b17dca382ccbfb08a853cffc79e2049607f95455cc44fa",
		Payload:       models.Payload(`{"schemaVersion":2,"mediaType":"...","config":{}}`),
	}
	err := s.Create(suite.ctx, m)

	require.NoError(t, err)
	require.NotEmpty(t, m.ID)
	require.NotEmpty(t, m.CreatedAt)
}

func TestManifestStore_Create_NonUniqueDigestFails(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)

	// see testdata/fixtures/manifests.sql
	m := &models.Manifest{
		RepositoryID:  3,
		SchemaVersion: 2,
		MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
		Digest:        "sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155",
		Payload:       models.Payload(`{"schemaVersion":2,"mediaType":"...","config":{}}`),
	}
	err := s.Create(suite.ctx, m)
	require.Error(t, err)
}

func TestManifestStore_AssociateLayerBlob(t *testing.T) {
	reloadManifestFixtures(t)
	require.NoError(t, testutil.TruncateTables(suite.db, testutil.LayersTable))

	s := datastore.NewManifestStore(suite.db)
	m := &models.Manifest{RepositoryID: 3, ID: 1}
	b := &models.Blob{
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Digest:    "sha256:f01256086224ded321e042e74135d72d5f108089a1cda03ab4820dfc442807c1",
		Size:      109,
	}

	err := s.AssociateLayerBlob(suite.ctx, m, b)
	require.NoError(t, err)

	bb, err := s.LayerBlobs(suite.ctx, m)
	require.NoError(t, err)

	// see testdata/fixtures/layers.sql
	local := bb[0].CreatedAt.Location()
	expected := models.Blobs{
		{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:f01256086224ded321e042e74135d72d5f108089a1cda03ab4820dfc442807c1",
			Size:      109,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:06:32.856423", local),
		},
	}
	require.Equal(t, expected, bb)
}

func TestManifestStore_AssociateManifest(t *testing.T) {
	reloadManifestFixtures(t)
	require.NoError(t, testutil.TruncateTables(suite.db, testutil.ManifestReferencesTable))

	s := datastore.NewManifestStore(suite.db)

	// see testdata/fixtures/manifest_references.sql
	ml := &models.Manifest{RepositoryID: 4, ID: 7}
	m := &models.Manifest{RepositoryID: 4, ID: 4}
	err := s.AssociateManifest(suite.ctx, ml, m)
	require.NoError(t, err)

	mm, err := s.References(suite.ctx, ml)
	require.NoError(t, err)

	var assocManifestIDs []int64
	for _, m := range mm {
		assocManifestIDs = append(assocManifestIDs, m.ID)
	}
	require.Contains(t, assocManifestIDs, int64(4))
}

func TestManifestStore_AssociateManifest_AlreadyAssociatedDoesNotFail(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)

	// see testdata/fixtures/manifest_references.sql
	ml := &models.Manifest{RepositoryID: 4, ID: 7}
	m := &models.Manifest{RepositoryID: 4, ID: 3}
	err := s.AssociateManifest(suite.ctx, ml, m)
	require.NoError(t, err)
}

func TestManifestStore_AssociateManifest_WithItselfFails(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)

	ml := &models.Manifest{ID: 5}
	m := &models.Manifest{ID: 5}
	err := s.AssociateManifest(suite.ctx, ml, m)
	require.Error(t, err)
}

func TestManifestStore_AssociateManifest_NestedLists(t *testing.T) {
	reloadManifestFixtures(t)
	require.NoError(t, testutil.TruncateTables(suite.db, testutil.ManifestReferencesTable))

	s := datastore.NewManifestStore(suite.db)

	// create a new temporary manifest list
	tmp := &models.Manifest{
		RepositoryID:  7,
		SchemaVersion: 2,
		MediaType:     "application/vnd.docker.distribution.manifest.list.v2+json",
		Digest:        "sha256:86b163863b462eadc1b17dca382ccbfb08a853cffc79e2049607f95455cc44fa",
		Payload:       models.Payload(`{"schemaVersion":2,"mediaType":"...","config":{}}`),
	}
	err := s.Create(suite.ctx, tmp)
	require.NoError(t, err)
	require.NotEmpty(t, tmp.ID)

	// see testdata/fixtures/manifests.sql
	ml1 := &models.Manifest{RepositoryID: 7, ID: 8}
	ml2 := &models.Manifest{RepositoryID: 7, ID: tmp.ID}
	err = s.AssociateManifest(suite.ctx, ml1, ml2)
	require.NoError(t, err)

	mm, err := s.References(suite.ctx, ml1)
	require.NoError(t, err)

	var assocManifestIDs []int64
	for _, m := range mm {
		assocManifestIDs = append(assocManifestIDs, m.ID)
	}
	require.Contains(t, assocManifestIDs, tmp.ID)
}

func TestManifestStore_DissociateManifest(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)
	ml := &models.Manifest{ID: 6}
	m := &models.Manifest{ID: 1}

	err := s.DissociateManifest(suite.ctx, ml, m)
	require.NoError(t, err)

	mm, err := s.References(suite.ctx, ml)
	require.NoError(t, err)

	// see testdata/fixtures/manifest_references.sql
	var manifestIDs []int64
	for _, m := range mm {
		manifestIDs = append(manifestIDs, m.ID)
	}
	require.NotContains(t, manifestIDs, 1)
}

func TestManifestStore_DissociateManifest_NotAssociatedDoesNotFail(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)
	ml := &models.Manifest{ID: 6}
	m := &models.Manifest{ID: 3}

	err := s.DissociateManifest(suite.ctx, ml, m)
	require.NoError(t, err)
}

func TestManifestStore_AssociateLayerBlob_AlreadyAssociatedDoesNotFail(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)

	// see testdata/fixtures/layers.sql
	m := &models.Manifest{RepositoryID: 3, ID: 1}
	b := &models.Blob{
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Digest:    "sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9",
		Size:      2802957,
	}
	err := s.AssociateLayerBlob(suite.ctx, m, b)
	require.NoError(t, err)
}

func TestManifestStore_DissociateLayerBlob(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)
	m := &models.Manifest{RepositoryID: 3, ID: 1}
	b := &models.Blob{
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Digest:    "sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9",
		Size:      2802957,
	}
	err := s.DissociateLayerBlob(suite.ctx, m, b)
	require.NoError(t, err)

	bb, err := s.LayerBlobs(suite.ctx, m)
	require.NoError(t, err)

	// see testdata/fixtures/layers.sql
	local := bb[0].CreatedAt.Location()
	unexpected := models.Blobs{
		{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9",
			Size:      2802957,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:05:35.338639", local),
		},
	}
	require.NotContains(t, bb, unexpected)
}

func TestManifestStore_DissociateLayerBlob_NotAssociatedDoesNotFail(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)
	m := &models.Manifest{RepositoryID: 3, ID: 1}
	b := &models.Blob{Digest: "sha256:c4039fd85dccc8e267c98447f8f1b27a402dbb4259d86586f4097acb5e6634af"}

	err := s.DissociateLayerBlob(suite.ctx, m, b)
	require.NoError(t, err)
}
