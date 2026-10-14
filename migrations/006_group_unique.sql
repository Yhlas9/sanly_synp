-- Bir mugallymda şol bir atly iki topar bolmaly däl: olar žurnaly bölýär
-- we haýsysyna seretmelidigi düşnüksiz bolýar.
--
-- Öň döredilen gaýtalanmalar birleşdirilýär. Tertip möhüm: ilki ýetmeýän
-- atlar esasy topara göçürilýär, soň tabşyrylan işler täze ýazga baglanýar,
-- diňe şondan soň artykmaç ýazgylar pozulýar — şeýdip žurnalyň taryhy ýitmeýär.
do $$
declare
  dup record;
begin
  for dup in
    select teacher_id, name, min(id) as keep_id, array_agg(id) as ids
      from groups
     group by teacher_id, name
    having count(*) > 1
  loop
    -- 1. Esasy toparda ýok atlary goşýarys (gaýtalanmalar aýrylýar).
    insert into students (group_id, full_name)
    select dup.keep_id, min(s.full_name)
      from students s
     where s.group_id = any(dup.ids)
       and s.group_id <> dup.keep_id
     group by lower(s.full_name)
    on conflict (group_id, full_name) do nothing;

    -- 2. Öň tabşyrylan işleri saklanýan okuwça baglaýarys.
    update participants p
       set student_id = keep.id
      from students old
      join students keep
        on keep.group_id = dup.keep_id
       and lower(keep.full_name) = lower(old.full_name)
     where p.student_id = old.id
       and old.group_id = any(dup.ids)
       and old.group_id <> dup.keep_id;

    -- 3. Indi artykmaç ýazgylary pozup bolýar.
    delete from students
     where group_id = any(dup.ids) and group_id <> dup.keep_id;

    update sessions set group_id = dup.keep_id
     where group_id = any(dup.ids) and group_id <> dup.keep_id;

    delete from groups where id = any(dup.ids) and id <> dup.keep_id;
  end loop;
end $$;

create unique index if not exists uq_groups_teacher_name on groups (teacher_id, name);
