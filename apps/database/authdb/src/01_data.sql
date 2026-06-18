INSERT INTO auth.roles (role_name, role_description, created_by_user_id, created_time, modified_by_user_id, modified_time)
VALUES ('ADMIN', 'Administrator role with full access.', 1, CURRENT_TIMESTAMP, 1, CURRENT_TIMESTAMP);

INSERT INTO auth.roles (role_name, role_description, created_by_user_id, created_time, modified_by_user_id, modified_time)
VALUES ('MANAGER', 'Manager role with limited access.', 1, CURRENT_TIMESTAMP, 1, CURRENT_TIMESTAMP);

INSERT INTO auth.roles (role_name, role_description, created_by_user_id, created_time, modified_by_user_id, modified_time)
VALUES ('USER', 'Standard user role with basic access.', 1, CURRENT_TIMESTAMP, 1, CURRENT_TIMESTAMP);

INSERT INTO auth.users (email_address, password, created_by_user_id, created_time, modified_by_user_id, modified_time)
VALUES ('system@jdwlabs.com', '$2a$10$v5e3IStWiGNj1tmB8WWoguoVQHFThHyMlqaOJP1vrNyrrcZfwDRBa', 1, CURRENT_TIMESTAMP, 1, CURRENT_TIMESTAMP);

-- Resolve ids by natural key (email, role_name) so the grant survives whatever
-- values the identity sequences assign instead of assuming 1/1,2,3.
INSERT INTO auth.users_roles (user_id, role_id, created_by_user_id, created_time)
SELECT u.user_id, r.role_id, u.user_id, CURRENT_TIMESTAMP
FROM auth.users u
CROSS JOIN auth.roles r
WHERE u.email_address = 'system@jdwlabs.com'
  AND r.role_name IN ('ADMIN', 'MANAGER', 'USER');
