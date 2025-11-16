-- Minimal deterministic seed for review
insert into teams(id,name) values
  ('11111111-1111-1111-1111-111111111111','core'),
  ('22222222-2222-2222-2222-222222222222','ml')
on conflict do nothing;

insert into users(id,username,is_active) values
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','alice',true),
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','bob',true),
  ('cccccccc-cccc-cccc-cccc-cccccccccccc','carol',false)
on conflict do nothing;

insert into team_members(team_id,user_id) values
  ('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'),
  ('11111111-1111-1111-1111-111111111111','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'),
  ('11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc')
on conflict do nothing;
