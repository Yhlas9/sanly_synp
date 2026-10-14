#!/bin/bash
# Görkezmeden öň synag maglumatlaryny arassalamak.
# Nusgalyk sapaklar we soraglar galýar, diňe geçirilen sapaklar pozulýar.
set -e
cd "$(dirname "$0")"
export PATH="/opt/homebrew/opt/postgresql@16/bin:$PATH"

psql -q -d sanly_synp <<'SQL'
delete from sessions;
delete from groups;
delete from teachers where login not in ('mugallym', 'admin');
delete from lessons where title like '%(nusga)%' or title like '%Synag sapagy%'
                       or title like '%Mugallymyň öz sapagy%';
delete from tests where title like '%Ýük barlagy%' or title like '%Garyşdyrylan%'
                     or title like '%Mugallymyň öz synagy%' or title like 'Retake=%';
SQL

echo "Synag maglumatlary arassalandy. Nusgalyk mazmun ýerinde:"
psql -q -d sanly_synp -c "select s.name_tk as ders, count(distinct l.id) as sapak, count(distinct q.id) as sorag
  from subjects s left join lessons l on l.subject_id = s.id
  left join questions q on q.lesson_id = l.id
  group by s.name_tk, s.sort_order order by s.sort_order;"
