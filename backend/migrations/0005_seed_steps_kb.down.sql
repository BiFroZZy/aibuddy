DELETE FROM kb_chunks WHERE document_id IN (SELECT id FROM kb_documents WHERE source LIKE 'test-data/%');
DELETE FROM kb_documents WHERE source LIKE 'test-data/%';
DELETE FROM intern_steps WHERE step_id IN (SELECT id FROM scenario_steps WHERE scenario_id = 'bbbbbbbb-0000-0000-0000-000000000001');
DELETE FROM scenario_steps WHERE scenario_id = 'bbbbbbbb-0000-0000-0000-000000000001';
DELETE FROM scenarios WHERE id = 'bbbbbbbb-0000-0000-0000-000000000001';
DELETE FROM teams WHERE id = '22222222-2222-2222-2222-222222222222';