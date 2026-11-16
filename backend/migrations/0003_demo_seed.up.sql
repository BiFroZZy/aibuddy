INSERT INTO teams (id, name) VALUES
    ('11111111-1111-1111-1111-111111111111', 'Backend (тестовая команда)')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, login, full_name, role, team_id, mentor_id, manager_id) VALUES
    ('c390da66-1700-4705-b5b3-d24c40914e2d', 'manager', 'Мария Руководитель', 'manager',  '11111111-1111-1111-1111-111111111111', NULL,       NULL),
    ('abff9c90-827e-4e33-b9ec-8c1082a75f63',  'mentor',   'Михаил Наставник',  'mentor',  '11111111-1111-1111-1111-111111111111', NULL,       'c390da66-1700-4705-b5b3-d24c40914e2d'),
    ('fb8a5a61-d41c-464d-b9ee-fe07b3745136',      'hr',           'Ольга HR',   'hr_bp',   NULL,      NULL,       NULL),
    ('a2be6219-ca2c-4f3b-bcb1-1e6c07622624', 'intern1',        'Анна Стажёр',  'intern',  '11111111-1111-1111-1111-111111111111', 'abff9c90-827e-4e33-b9ec-8c1082a75f63', 'c390da66-1700-4705-b5b3-d24c40914e2d'),
    ('28a82a11-925c-4783-a32e-426278bfec83', 'intern2',       'Борис Стажёр',  'intern',  '11111111-1111-1111-1111-111111111111', 'abff9c90-827e-4e33-b9ec-8c1082a75f63', 'c390da66-1700-4705-b5b3-d24c40914e2d')
ON CONFLICT (id) DO NOTHING;

INSERT INTO mock_auth_tokens (token, user_id) VALUES
    ('dev-intern-1', 'a2be6219-ca2c-4f3b-bcb1-1e6c07622624'),
    ('dev-intern-2', '28a82a11-925c-4783-a32e-426278bfec83'),
    ('dev-mentor',   'abff9c90-827e-4e33-b9ec-8c1082a75f63'),
    ('dev-manager',  'c390da66-1700-4705-b5b3-d24c40914e2d'),
    ('dev-hr',       'fb8a5a61-d41c-464d-b9ee-fe07b3745136')
ON CONFLICT (token) DO NOTHING;