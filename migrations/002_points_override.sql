-- Учитель может переопределить балл вопроса внутри конкретного теста,
-- не трогая банк вопросов.
alter table test_questions
    add column if not exists points_override numeric(6, 2);
