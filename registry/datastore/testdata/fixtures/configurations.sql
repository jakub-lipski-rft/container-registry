INSERT INTO "configurations"("id", "blob_id", "payload", "created_at")
VALUES (1, 8, convert_to(
        '{"architecture":"amd64","config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh"],"ArgsEscaped":true,"Image":"sha256:e7d92cdc71feacf90708cb59182d0df1b911f8ae022d29e8e95d75ca6a99776a","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"container":"7980908783eb05384926afb5ffad45856f65bc30029722a4be9f1eb3661e9c5e","container_config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh","-c","echo \"1\" \u003e /data"],"Image":"sha256:e7d92cdc71feacf90708cb59182d0df1b911f8ae022d29e8e95d75ca6a99776a","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"created":"2020-03-02T12:21:53.8027967Z","docker_version":"19.03.5","history":[{"created":"2020-01-18T01:19:37.02673981Z","created_by":"/bin/sh -c #(nop) ADD file:e69d441d729412d24675dcd33e04580885df99981cec43de8c9b24015313ff8e in / "},{"created":"2020-01-18T01:19:37.187497623Z","created_by":"/bin/sh -c #(nop)  CMD [\"/bin/sh\"]","empty_layer":true},{"created":"2020-03-02T12:21:53.8027967Z","created_by":"/bin/sh -c echo \"1\" \u003e /data"}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:5216338b40a7b96416b8b9858974bbe4acc3096ee60acbc4dfb1ee02aecceb10","sha256:99cb4c5d9f96432a00201f4b14c058c6235e563917ba7af8ed6c4775afa5780f"]}}',
        'UTF8'), E'2020-03-02 17:56:26.573726+00'),
       (2, 9, convert_to(
               '{"architecture":"amd64","config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh"],"ArgsEscaped":true,"Image":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"container":"cb78c8a8058712726096a7a8f80e6a868ffb514a07f4fef37639f42d99d997e4","container_config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh","-c","echo \"2\" \u003e\u003e /data"],"Image":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"created":"2020-03-02T12:24:16.7039823Z","docker_version":"19.03.5","history":[{"created":"2020-01-18T01:19:37.02673981Z","created_by":"/bin/sh -c #(nop) ADD file:e69d441d729412d24675dcd33e04580885df99981cec43de8c9b24015313ff8e in / "},{"created":"2020-01-18T01:19:37.187497623Z","created_by":"/bin/sh -c #(nop)  CMD [\"/bin/sh\"]","empty_layer":true},{"created":"2020-03-02T12:21:53.8027967Z","created_by":"/bin/sh -c echo \"1\" \u003e /data"},{"created":"2020-03-02T12:24:16.7039823Z","created_by":"/bin/sh -c echo \"2\" \u003e\u003e /data"}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:5216338b40a7b96416b8b9858974bbe4acc3096ee60acbc4dfb1ee02aecceb10","sha256:99cb4c5d9f96432a00201f4b14c058c6235e563917ba7af8ed6c4775afa5780f","sha256:6322c07f5c6ad456f64647993dfc44526f4548685ee0f3d8f03534272b3a06d8"]}}',
               'UTF8'), E'2020-03-02 17:57:23.405516+00'),
       (3, 10, convert_to(
               '{"architecture":"amd64","config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"ExposedPorts":{"80/tcp":{}},"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin","NGINX_VERSION=1.17.8","NJS_VERSION=0.3.8","PKG_RELEASE=1~buster"],"Cmd":["nginx","-g","daemon off;"],"ArgsEscaped":true,"Image":"sha256:a1523e859360df9ffe2b31a8270f5e16422609fe138c1636383efdc34b9ea2d6","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":{"maintainer":"NGINX Docker Maintainers \u003cdocker-maint@nginx.com\u003e"},"StopSignal":"SIGTERM"},"container":"9a24d3f0d5ca79fceaef1956a91e0ba05b2e924295b8b0ec439db5a6bd491dda","container_config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"ExposedPorts":{"80/tcp":{}},"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin","NGINX_VERSION=1.17.8","NJS_VERSION=0.3.8","PKG_RELEASE=1~buster"],"Cmd":["/bin/sh","-c","echo \"1\" \u003e /data"],"Image":"sha256:a1523e859360df9ffe2b31a8270f5e16422609fe138c1636383efdc34b9ea2d6","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":{"maintainer":"NGINX Docker Maintainers \u003cdocker-maint@nginx.com\u003e"},"StopSignal":"SIGTERM"},"created":"2020-03-02T12:34:49.9572024Z","docker_version":"19.03.5","history":[{"created":"2020-02-26T00:37:39.301941924Z","created_by":"/bin/sh -c #(nop) ADD file:e5a364615e0f6961626089c7d658adbf8c8d95b3ae95a390a8bb33875317d434 in / "},{"created":"2020-02-26T00:37:39.539684396Z","created_by":"/bin/sh -c #(nop)  CMD [\"bash\"]","empty_layer":true},{"created":"2020-02-26T20:01:52.907016299Z","created_by":"/bin/sh -c #(nop)  LABEL maintainer=NGINX Docker Maintainers \u003cdocker-maint@nginx.com\u003e","empty_layer":true},{"created":"2020-02-26T20:01:53.114563769Z","created_by":"/bin/sh -c #(nop)  ENV NGINX_VERSION=1.17.8","empty_layer":true},{"created":"2020-02-26T20:01:53.28669526Z","created_by":"/bin/sh -c #(nop)  ENV NJS_VERSION=0.3.8","empty_layer":true},{"created":"2020-02-26T20:01:53.470888291Z","created_by":"/bin/sh -c #(nop)  ENV PKG_RELEASE=1~buster","empty_layer":true},{"created":"2020-02-26T20:02:14.311730686Z","created_by":"/bin/sh -c set -x     \u0026\u0026 addgroup --system --gid 101 nginx     \u0026\u0026 adduser --system --disabled-login --ingroup nginx --no-create-home --home /nonexistent --gecos \"nginx user\" --shell /bin/false --uid 101 nginx     \u0026\u0026 apt-get update     \u0026\u0026 apt-get install --no-install-recommends --no-install-suggests -y gnupg1 ca-certificates     \u0026\u0026     NGINX_GPGKEY=573BFD6B3D8FBC641079A6ABABF5BD827BD9BF62;     found='''';     for server in         ha.pool.sks-keyservers.net         hkp://keyserver.ubuntu.com:80         hkp://p80.pool.sks-keyservers.net:80         pgp.mit.edu     ; do         echo \"Fetching GPG key $NGINX_GPGKEY from $server\";         apt-key adv --keyserver \"$server\" --keyserver-options timeout=10 --recv-keys \"$NGINX_GPGKEY\" \u0026\u0026 found=yes \u0026\u0026 break;     done;     test -z \"$found\" \u0026\u0026 echo \u003e\u00262 \"error: failed to fetch GPG key $NGINX_GPGKEY\" \u0026\u0026 exit 1;     apt-get remove --purge --auto-remove -y gnupg1 \u0026\u0026 rm -rf /var/lib/apt/lists/*     \u0026\u0026 dpkgArch=\"$(dpkg --print-architecture)\"     \u0026\u0026 nginxPackages=\"         nginx=${NGINX_VERSION}-${PKG_RELEASE}         nginx-module-xslt=${NGINX_VERSION}-${PKG_RELEASE}         nginx-module-geoip=${NGINX_VERSION}-${PKG_RELEASE}         nginx-module-image-filter=${NGINX_VERSION}-${PKG_RELEASE}         nginx-module-njs=${NGINX_VERSION}.${NJS_VERSION}-${PKG_RELEASE}     \"     \u0026\u0026 case \"$dpkgArch\" in         amd64|i386)             echo \"deb https://nginx.org/packages/mainline/debian/ buster nginx\" \u003e\u003e /etc/apt/sources.list.d/nginx.list             \u0026\u0026 apt-get update             ;;         *)             echo \"deb-src https://nginx.org/packages/mainline/debian/ buster nginx\" \u003e\u003e /etc/apt/sources.list.d/nginx.list                         \u0026\u0026 tempDir=\"$(mktemp -d)\"             \u0026\u0026 chmod 777 \"$tempDir\"                         \u0026\u0026 savedAptMark=\"$(apt-mark showmanual)\"                         \u0026\u0026 apt-get update             \u0026\u0026 apt-get build-dep -y $nginxPackages             \u0026\u0026 (                 cd \"$tempDir\"                 \u0026\u0026 DEB_BUILD_OPTIONS=\"nocheck parallel=$(nproc)\"                     apt-get source --compile $nginxPackages             )                         \u0026\u0026 apt-mark showmanual | xargs apt-mark auto \u003e /dev/null             \u0026\u0026 { [ -z \"$savedAptMark\" ] || apt-mark manual $savedAptMark; }                         \u0026\u0026 ls -lAFh \"$tempDir\"             \u0026\u0026 ( cd \"$tempDir\" \u0026\u0026 dpkg-scanpackages . \u003e Packages )             \u0026\u0026 grep ''^Package: '' \"$tempDir/Packages\"             \u0026\u0026 echo \"deb [ trusted=yes ] file://$tempDir ./\" \u003e /etc/apt/sources.list.d/temp.list             \u0026\u0026 apt-get -o Acquire::GzipIndexes=false update             ;;     esac         \u0026\u0026 apt-get install --no-install-recommends --no-install-suggests -y                         $nginxPackages                         gettext-base     \u0026\u0026 apt-get remove --purge --auto-remove -y ca-certificates \u0026\u0026 rm -rf /var/lib/apt/lists/* /etc/apt/sources.list.d/nginx.list         \u0026\u0026 if [ -n \"$tempDir\" ]; then         apt-get purge -y --auto-remove         \u0026\u0026 rm -rf \"$tempDir\" /etc/apt/sources.list.d/temp.list;     fi"},{"created":"2020-02-26T20:02:15.146823517Z","created_by":"/bin/sh -c ln -sf /dev/stdout /var/log/nginx/access.log     \u0026\u0026 ln -sf /dev/stderr /var/log/nginx/error.log"},{"created":"2020-02-26T20:02:15.335986561Z","created_by":"/bin/sh -c #(nop)  EXPOSE 80","empty_layer":true},{"created":"2020-02-26T20:02:15.543209017Z","created_by":"/bin/sh -c #(nop)  STOPSIGNAL SIGTERM","empty_layer":true},{"created":"2020-02-26T20:02:15.724396212Z","created_by":"/bin/sh -c #(nop)  CMD [\"nginx\" \"-g\" \"daemon off;\"]","empty_layer":true},{"created":"2020-03-02T12:34:49.9572024Z","created_by":"/bin/sh -c echo \"1\" \u003e /data"}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:f2cb0ecef392f2a630fa1205b874ab2e2aedf96de04d0b8838e4e728e28142da","sha256:fe08d5d042ab93bee05f9cda17f1c57066e146b0704be2ff755d14c25e6aa5e8","sha256:318be7aea8fc62d5910cca0d49311fa8d95502c90e2a91b7a4d78032a670b644","sha256:ca5cd87c6bf8376275e0bf32cd7139ed17dd69ef28bda9ba15d07475b147f931"]}}',
               'UTF8'), E'2020-03-02 17:57:23.405516+00');