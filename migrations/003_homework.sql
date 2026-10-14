-- Öý işi: synag janly sapaga bagly bolman, bellenen möhlete çenli açyk durýar.
-- Okuwçy özüne amatly wagtda girip, işini ýerine ýetirýär.
alter table sessions
    add column if not exists mode      text not null default 'live', -- live | homework
    add column if not exists opens_at  timestamptz,
    add column if not exists closes_at timestamptz;

-- Her okuwçynyň öz möhleti işe başlan pursatyndan hasaplanýar.
alter table participants
    add column if not exists started_at timestamptz;

create index if not exists idx_sessions_mode on sessions (mode, closes_at);
