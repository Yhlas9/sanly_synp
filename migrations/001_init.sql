-- Sanly Synp — oflaýn okuw platformasy
-- Schema: чистый SQL, без ORM.

create table if not exists teachers (
    id            bigserial primary key,
    login         text        not null unique,
    password_hash text        not null,
    full_name     text        not null,
    role          text        not null default 'teacher',  -- teacher | admin
    created_at    timestamptz not null default now()
);

create table if not exists subjects (
    id         bigserial primary key,
    code       text not null unique,   -- matematika | fizika | himiya | biologiya | informatika
    name_tk    text not null,
    sort_order int  not null default 0
);

create table if not exists groups (
    id         bigserial primary key,
    teacher_id bigint      not null references teachers (id) on delete cascade,
    name       text        not null,
    created_at timestamptz not null default now()
);

-- Урок: текст, формулы, интерактивная демонстрация, практическое задание
create table if not exists lessons (
    id             bigserial primary key,
    teacher_id     bigint      not null references teachers (id) on delete cascade,
    subject_id     bigint      not null references subjects (id),
    title          text        not null,
    topic          text        not null default '',
    goal           text        not null default '',
    body_md        text        not null default '',
    demo_kind      text        not null default '',  -- function_plot|circuit|reaction|cell|algorithm
    demo_config    jsonb       not null default '{}'::jsonb,
    practical_task text        not null default '',
    created_at     timestamptz not null default now(),
    updated_at     timestamptz not null default now()
);

-- Банк вопросов. answer_key НИКОГДА не уходит на устройство ученика во время теста.
create table if not exists questions (
    id          bigserial primary key,
    lesson_id   bigint references lessons (id) on delete set null,
    subject_id  bigint not null references subjects (id),
    kind        text   not null,  -- choice|short|numeric|match|order|open|practical
    prompt      text   not null,
    payload     jsonb  not null default '{}'::jsonb,
    answer_key  jsonb  not null default '{}'::jsonb,
    points      numeric(6, 2) not null default 1,
    hint        text   not null default '',
    explanation text   not null default '',
    position    int    not null default 0,
    created_at  timestamptz not null default now()
);

create table if not exists tests (
    id                bigserial primary key,
    teacher_id        bigint not null references teachers (id) on delete cascade,
    subject_id        bigint not null references subjects (id),
    title             text   not null,
    duration_sec      int    not null default 600,
    shuffle_questions boolean not null default false,
    shuffle_options   boolean not null default false,
    allow_retake      boolean not null default false,
    results_policy    text   not null default 'after_close', -- immediately|after_close|manual
    created_at        timestamptz not null default now()
);

create table if not exists test_questions (
    test_id     bigint not null references tests (id) on delete cascade,
    question_id bigint not null references questions (id) on delete cascade,
    position    int    not null default 0,
    primary key (test_id, question_id)
);

-- Живое занятие. Срок теста считает СЕРВЕР: часы на устройстве ученика не влияют.
create table if not exists sessions (
    id             bigserial primary key,
    teacher_id     bigint      not null references teachers (id) on delete cascade,
    lesson_id      bigint references lessons (id) on delete set null,
    test_id        bigint references tests (id) on delete set null,
    group_id       bigint references groups (id) on delete set null,
    join_code      text        not null unique,
    state          text        not null default 'lobby',  -- lobby|lesson|test|closed
    join_open      boolean     not null default true,
    test_started_at timestamptz,
    test_deadline   timestamptz,
    -- Содержание теста фиксируется на момент старта: правки в банке вопросов
    -- не меняют результаты уже сданных работ.
    frozen_questions jsonb,
    created_at     timestamptz not null default now()
);

create table if not exists participants (
    id           bigserial primary key,
    session_id   bigint      not null references sessions (id) on delete cascade,
    display_name text        not null,
    token        text        not null unique,
    connected    boolean     not null default false,
    last_seen    timestamptz,
    joined_at    timestamptz not null default now(),
    unique (session_id, display_name)
);

-- client_key даёт идемпотентность: один ответ не засчитывается дважды
-- даже при повторной отправке после обрыва сети.
create table if not exists answers (
    id              bigserial primary key,
    session_id      bigint not null references sessions (id) on delete cascade,
    participant_id  bigint not null references participants (id) on delete cascade,
    question_id     bigint not null references questions (id) on delete cascade,
    client_key      text   not null,
    value           jsonb  not null default '{}'::jsonb,
    status          text   not null default 'saved',  -- saved|submitted
    is_correct      boolean,
    auto_score      numeric(6, 2),
    manual_score    numeric(6, 2),
    final_score     numeric(6, 2),
    needs_manual    boolean not null default false,
    teacher_comment text    not null default '',
    created_at      timestamptz not null default now(),
    updated_at      timestamptz not null default now(),
    unique (participant_id, question_id)
);

-- Корректировка оценки: сохраняются старая, новая и причина.
create table if not exists grade_adjustments (
    id         bigserial primary key,
    answer_id  bigint      not null references answers (id) on delete cascade,
    old_score  numeric(6, 2),
    new_score  numeric(6, 2),
    reason     text        not null default '',
    created_at timestamptz not null default now()
);

create table if not exists submissions (
    id             bigserial primary key,
    session_id     bigint not null references sessions (id) on delete cascade,
    participant_id bigint not null references participants (id) on delete cascade,
    status         text   not null default 'in_progress', -- in_progress|submitted|late
    submitted_at   timestamptz,
    extra_time_sec int    not null default 0,
    unique (session_id, participant_id)
);

create index if not exists idx_lessons_subject      on lessons (subject_id);
create index if not exists idx_questions_subject    on questions (subject_id);
create index if not exists idx_questions_lesson     on questions (lesson_id);
create index if not exists idx_participants_session on participants (session_id);
create index if not exists idx_answers_session      on answers (session_id);
create index if not exists idx_answers_participant  on answers (participant_id);
