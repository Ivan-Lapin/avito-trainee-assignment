create table if not exists users(
  id uuid primary key,
  username text not null unique,
  is_active boolean not null default true
);

create table if not exists teams(
  id uuid primary key,
  name text not null unique
);

create table if not exists team_members(
  team_id uuid not null references teams(id) on delete cascade,
  user_id uuid not null references users(id) on delete cascade,
  primary key(team_id, user_id)
);

create table if not exists pull_requests(
  id uuid primary key,
  title text not null,
  author_id uuid not null references users(id),
  status text not null check (status in ('OPEN','MERGED')),
  created_at timestamptz not null default now(),
  merged_at timestamptz
);

create index if not exists idx_pr_status on pull_requests(status);
create index if not exists idx_pr_author on pull_requests(author_id);

create table if not exists pr_reviewers(
  pr_id uuid not null references pull_requests(id) on delete cascade,
  reviewer_id uuid not null references users(id),
  primary key(pr_id, reviewer_id)
);

create index if not exists idx_pr_reviewers_pr on pr_reviewers(pr_id);
