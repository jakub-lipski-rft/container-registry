INSERT INTO "repositories"("id", "name", "path", "parent_id", "created_at", "deleted_at")
VALUES (1, 'gitlab-org', 'gitlab-org', NULL, '2020-03-02 17:47:39.849864', NULL),
       (2, 'gitlab-test', 'gitlab-org/gitlab-test', 1, '2020-03-02 17:47:40.866312', NULL),
       (3, 'backend', 'gitlab-org/gitlab-test/backend', 2, '2020-03-02 17:42:12.566212', NULL),
       (4, 'frontend', 'gitlab-org/gitlab-test/frontend', 2, '2020-03-02 17:43:39.476421', NULL);