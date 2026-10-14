-- Amaly iş: okuwçy tejribäniň netijesini faýl ýa-da surat görnüşinde tabşyryp biler
-- (ТЗ, 6-njy bölüm). Faýlyň özi diskde, maglumat binýadynda diňe ýazgysy saklanýar.
create table if not exists uploads (
    id             text primary key,
    session_id     bigint      not null references sessions (id) on delete cascade,
    participant_id bigint      not null references participants (id) on delete cascade,
    file_name      text        not null,
    content_type   text        not null default '',
    size_bytes     bigint      not null default 0,
    stored_path    text        not null,
    created_at     timestamptz not null default now()
);

create index if not exists idx_uploads_participant on uploads (participant_id);
