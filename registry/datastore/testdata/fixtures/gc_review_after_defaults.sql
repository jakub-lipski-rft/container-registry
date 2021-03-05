INSERT INTO gc_review_after_defaults (event, value)
VALUES ('blob_upload', interval '1 day'),
       ('manifest_upload', interval '7 days'),
       ('manifest_delete', interval '1 hour'),
       ('layer_delete', interval '16 hours'),
       ('manifest_list_delete', interval '1 minute'),
       ('tag_delete', interval '21 minute'),
       ('tag_switch', interval '0');