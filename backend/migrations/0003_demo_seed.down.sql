DELETE FROM intern_steps
WHERE user_id IN (
    'c390da66-1700-4705-b5b3-d24c40914e2d',
    'abff9c90-827e-4e33-b9ec-8c1082a75f63',
    'fb8a5a61-d41c-464d-b9ee-fe07b3745136',
    'a2be6219-ca2c-4f3b-bcb1-1e6c07622624',
    '28a82a11-925c-4783-a32e-426278bfec83'
);
DELETE FROM action_log
WHERE user_id IN (
    'c390da66-1700-4705-b5b3-d24c40914e2d',
    'abff9c90-827e-4e33-b9ec-8c1082a75f63',
    'fb8a5a61-d41c-464d-b9ee-fe07b3745136',
    'a2be6219-ca2c-4f3b-bcb1-1e6c07622624',
    '28a82a11-925c-4783-a32e-426278bfec83'
);
DELETE FROM events
WHERE user_id IN (
    'c390da66-1700-4705-b5b3-d24c40914e2d',
    'abff9c90-827e-4e33-b9ec-8c1082a75f63',
    'fb8a5a61-d41c-464d-b9ee-fe07b3745136',
    'a2be6219-ca2c-4f3b-bcb1-1e6c07622624',
    '28a82a11-925c-4783-a32e-426278bfec83'
);
DELETE FROM conversations
WHERE user_id IN (
    'c390da66-1700-4705-b5b3-d24c40914e2d',
    'abff9c90-827e-4e33-b9ec-8c1082a75f63',
    'fb8a5a61-d41c-464d-b9ee-fe07b3745136',
    'a2be6219-ca2c-4f3b-bcb1-1e6c07622624',
    '28a82a11-925c-4783-a32e-426278bfec83'
);
DELETE FROM mock_auth_tokens
WHERE token IN ('dev-intern-1', 'dev-intern-2', 'dev-mentor', 'dev-manager', 'dev-hr');

DELETE FROM users
WHERE id IN (
    'c390da66-1700-4705-b5b3-d24c40914e2d',
    'abff9c90-827e-4e33-b9ec-8c1082a75f63',
    'fb8a5a61-d41c-464d-b9ee-fe07b3745136',
    'a2be6219-ca2c-4f3b-bcb1-1e6c07622624',
    '28a82a11-925c-4783-a32e-426278bfec83'
);

DELETE FROM teams WHERE id = 'b7f0a3c1-4a12-4a0d-9b21-2d7e4f8a1001';
