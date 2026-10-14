-- Her mugallym diňe öz materiallaryny görýär we üýtgedýär.
-- Nusgalyk mazmun (platforma bilen gelýän bäş ders) ählisine görünýär,
-- ýöne diňe okamak üçin: ony nusga alyp, öz adyňa uýgunlaşdyryp bolýar.
alter table lessons add column if not exists is_sample boolean not null default false;
alter table tests   add column if not exists is_sample boolean not null default false;

-- Başlangyç mazmun nusgalyk diýlip bellenýär: ol iň köne mugallyma degişli
-- we onuň soň döreden materiallaryndan tapawutlanmaly.
update lessons l set is_sample = true
 where l.teacher_id = (select min(id) from teachers)
   and l.created_at < now() - interval '1 second'
   and exists (select 1 from subjects s where s.id = l.subject_id)
   and l.demo_kind <> ''
   and not l.is_sample
   and l.title not like '%(nusga)%';

update tests t set is_sample = true
 where t.teacher_id = (select min(id) from teachers)
   and t.title like '%barlag%'
   and not t.is_sample;

create index if not exists idx_lessons_owner on lessons (teacher_id, is_sample);
create index if not exists idx_tests_owner   on tests (teacher_id, is_sample);
