create table if not exists system_traffic_probe_alerts (
  id integer primary key autoincrement,
  target_id integer not null references system_traffic_probe_targets(id) on delete cascade,
  result_id integer not null unique references system_traffic_probe_results(id) on delete cascade,
  severity text not null check (severity in ('warning', 'critical')),
  status text not null check (status in ('open', 'acknowledged', 'resolved')),
  reason text not null,
  detail_json text not null,
  opened_at text not null,
  acknowledged_at text,
  resolved_at text
);

create index if not exists idx_system_traffic_probe_alerts_status
  on system_traffic_probe_alerts(status, opened_at);

create index if not exists idx_system_traffic_probe_alerts_target
  on system_traffic_probe_alerts(target_id, status);
