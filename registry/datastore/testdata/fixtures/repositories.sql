INSERT INTO "repositories"("id", "name", "path", "parent_id", "created_at")
VALUES (1, 'gitlab-org', 'gitlab-org', NULL, '2020-03-02 17:47:39.849864+00'),
       (2, 'gitlab-test', 'gitlab-org/gitlab-test', 1, '2020-03-02 17:47:40.866312+00'),
       (3, 'backend', 'gitlab-org/gitlab-test/backend', 2, '2020-03-02 17:42:12.566212+00'),
       (4, 'frontend', 'gitlab-org/gitlab-test/frontend', 2, '2020-03-02 17:43:39.476421+00'),
       (5, 'a-test-group', 'a-test-group', NULL, '2020-06-08 16:01:39.476421+00'),
       (6, 'foo', 'a-test-group/foo', 5, '2020-06-08 16:01:39.476421+00'),
       (7, 'bar', 'a-test-group/bar', 5, '2020-06-08 16:01:39.476421+00');
