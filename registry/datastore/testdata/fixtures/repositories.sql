INSERT INTO "repositories"("id", "top_level_namespace_id", "name", "path", "parent_id", "created_at")
VALUES (1, 1, 'gitlab-org', 'gitlab-org', NULL, '2020-03-02 17:47:39.849864+00'),
       (2, 1, 'gitlab-test', 'gitlab-org/gitlab-test', 1, '2020-03-02 17:47:40.866312+00'),
       (3, 1, 'backend', 'gitlab-org/gitlab-test/backend', 2, '2020-03-02 17:42:12.566212+00'),
       (4, 1, 'frontend', 'gitlab-org/gitlab-test/frontend', 2, '2020-03-02 17:43:39.476421+00'),
       (5, 2, 'a-test-group', 'a-test-group', NULL, '2020-06-08 16:01:39.476421+00'),
       (6, 2, 'foo', 'a-test-group/foo', 5, '2020-06-08 16:01:39.476421+00'),
       (7, 2, 'bar', 'a-test-group/bar', 5, '2020-06-08 16:01:39.476421+00');
