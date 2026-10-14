-- Okuwçynyň öz hasaby: şahsy otaga girip, ähli işleriniň taryhyny görýär.
-- Hasap hökmany däl — topar sanawynda hasapsyz okuwçy hem galyp bilýär.
alter table students
    add column if not exists login         text,
    add column if not exists password_hash text,
    add column if not exists last_login_at  timestamptz;

create unique index if not exists uq_students_login
    on students (lower(login)) where login is not null;
