-- Topar sanawy we žurnal: okuwçy indi aýratyn sapaklaryň içinde däl-de,
-- toparda hemişelik ýazgy hökmünde ýaşaýar. Şonuň netijesinde onuň
-- ähli işleriniň taryhyny bir ýerde görüp bolýar.
create table if not exists students (
    id         bigserial primary key,
    group_id   bigint      not null references groups (id) on delete cascade,
    full_name  text        not null,
    created_at timestamptz not null default now(),
    unique (group_id, full_name)
);

-- Sapaga goşulan adam topar sanawyndaky okuwça baglanýar.
alter table participants
    add column if not exists student_id bigint references students (id) on delete set null;

create index if not exists idx_students_group      on students (group_id);
create index if not exists idx_participants_student on participants (student_id);
create index if not exists idx_sessions_group       on sessions (group_id);
