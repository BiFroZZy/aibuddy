INSERT INTO teams (id, name) VALUES
    ('backend', 'Backend (тестовая команда)')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, login, full_name, role, team_id, mentor_id, manager_id) VALUES
    ('u-manager',  'manager', 'Мария Руководитель', 'manager', 'backend', NULL,       NULL),
    ('u-mentor',   'mentor',  'Михаил Наставник',   'mentor',  'backend', NULL,       'u-manager'),
    ('u-hr',       'hr',      'Ольга HR',           'hr_bp',   NULL,      NULL,       NULL),
    ('u-intern-1', 'intern1', 'Анна Стажёр',        'intern',  'backend', 'u-mentor', 'u-manager'),
    ('u-intern-2', 'intern2', 'Борис Стажёр',       'intern',  'backend', 'u-mentor', 'u-manager')
ON CONFLICT (id) DO NOTHING;

INSERT INTO mock_auth_tokens (token, user_id) VALUES
    ('dev-intern-1', 'u-intern-1'),
    ('dev-intern-2', 'u-intern-2'),
    ('dev-mentor',   'u-mentor'),
    ('dev-manager',  'u-manager'),
    ('dev-hr',       'u-hr')
ON CONFLICT (token) DO NOTHING;