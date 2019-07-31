local test_pipeline = 'testing';

{
  test(os='linux', arch='amd64', version='')::
    local volumes = [{
      name: 'gopath',
      path: '/go',
    }];

    {
      kind: 'pipeline',
      name: test_pipeline,
      platform: {
        os: os,
        arch: arch,
        version: if std.length(version) > 0 then version,
      },
      steps: [
        {
          name: 'vet',
          image: 'golang:1.12',
          commands: [
            'go vet ./...',
          ],
          volumes: volumes,
        },
        {
          name: 'test',
          image: 'golang:1.12',
          commands: [
            'go test -cover ./...',
          ],
          volumes: volumes,
        },
      ],
      trigger: {
        ref: [
          'refs/heads/master',
          'refs/tags/**',
          'refs/pull/**',
        ],
      },
      volumes: [{
        name: 'gopath',
        temp: {},
      }],
    },

  build(name, os='linux', arch='amd64', version='')::
    local tag = os + '-' + arch;
    local suffix = std.strReplace(tag, '-', '.');
    local target = 'drone/' + std.splitLimit(name, '-', 1)[1];

    {
      kind: 'pipeline',
      name: tag,
      platform: {
        os: os,
        arch: arch,
        version: if std.length(version) > 0 then version,
      },
      steps: [
        {
          name: 'build',
          image: 'golang:1.12',
          environment: {
            CGO_ENABLED: '0',
          },
          commands: [
            'go build -v -a -tags netgo -o release/' + os + '/' + arch + '/' + name + ' ./cmd/' + name,
          ],
        },
        {
          name: 'dryrun',
          image: 'plugins/docker:' + tag,
          settings: {
            dry_run: true,
            tags: tag,
            dockerfile: 'docker/Dockerfile.' + suffix,
            repo: target,
            username: { from_secret: 'docker_username' },
            password: { from_secret: 'docker_password' },
          },
          when: {
            event: ['pull_request'],
          },
        },
        {
          name: 'publish',
          image: 'plugins/docker:' + tag,
          settings: {
            auto_tag: true,
            auto_tag_suffix: tag,
            dockerfile: 'docker/Dockerfile.' + suffix,
            repo: target,
            username: { from_secret: 'docker_username' },
            password: { from_secret: 'docker_password' },
          },
          when: {
            event: {
              exclude: ['pull_request'],
            },
          },
        },
        {
          name: 'tarball',
          image: 'golang:1.12',
          commands: [
            'tar -cvzf release/' + name + '_' + os + '_' + arch + '.tar.gz -C release/' + os + '/' + arch + ' ' + name,
            'sha256sum release/' + name + '_' + os + '_' + arch + '.tar.gz > release/' + name + '_' + os + '_' + arch + '.tar.gz.sha256'
          ],
          when: {
            event: ['tag'],
          },
        },
        {
          name: 'gpgsign',
          image: 'plugins/gpgsign',
          settings: {
            files: [
              'release/*.tar.gz',
              'release/*.tar.gz.sha256',
            ],
            key: { from_secret: 'gpgsign_key' },
            passphrase: { from_secret: 'gpgkey_passphrase' },
          },
          when: {
            event: ['tag'],
          },
        },
        {
          name: 'github',
          image: 'plugins/github-release',
          settings: {
            files: [
              'release/*.tar.gz',
              'release/*.tar.gz.sha256',
              'release/*.tar.gz.asc',
            ],
            token: { from_secret: 'github_token' },
          },
          when: {
            event: ['tag'],
          },
        },
      ],
      trigger: {
        ref: [
          'refs/heads/master',
          'refs/tags/**',
          'refs/pull/**',
        ],
      },
      depends_on: [test_pipeline],
    },

  notifications(os='linux', arch='amd64', version='', depends_on=[])::
    {
      kind: 'pipeline',
      name: 'notifications',
      platform: {
        os: os,
        arch: arch,
        version: if std.length(version) > 0 then version,
      },
      steps: [
        {
          name: 'manifest',
          image: 'plugins/manifest:latest',
          settings: {
            username: { from_secret: 'docker_username' },
            password: { from_secret: 'docker_password' },
            spec: 'docker/manifest.tmpl',
            ignore_missing: true,
          },
        },
        {
          name: 'microbadger',
          image: 'plugins/webhook',
          settings: {
            url: { from_secret: 'microbadger_url' },
          },
        },
      ],
      trigger: {
        ref: [
          'refs/heads/master',
          'refs/tags/**',
        ],
      },
      depends_on: depends_on,
    },
}
