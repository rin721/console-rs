alter table system_version_packages add column status text not null default 'draft';
alter table system_version_packages add column published_at text;
alter table system_version_packages add column retired_at text;

create table if not exists system_version_release_events (
  id integer primary key autoincrement,
  package_id integer not null,
  previous_active_id integer,
  action text not null,
  status text not null,
  reason text,
  created_at text not null,
  foreign key(package_id) references system_version_packages(id),
  foreign key(previous_active_id) references system_version_packages(id)
);

create index if not exists idx_system_version_release_events_package_id
  on system_version_release_events(package_id);
