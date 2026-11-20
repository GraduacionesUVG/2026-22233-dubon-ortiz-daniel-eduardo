INSERT INTO exercise_catalog (id, name_es, name_en, muscle_group) VALUES
('barbell_back_squat', 'Sentadilla con Barra', 'Barbell Back Squat', 'legs'),
('bulgarian_split_squat', 'Sentadilla Búlgara', 'Bulgarian Split Squat', 'legs'),
('romanian_deadlift', 'Peso Muerto Rumano', 'Romanian Deadlift', 'legs'),
('barbell_bench_press', 'Press de Banca con Barra', 'Barbell Bench Press', 'chest'),
('dumbbell_incline_press', 'Press Inclinado con Mancuernas', 'Incline Dumbbell Press', 'chest'),
('pull_up', 'Dominadas', 'Pull Ups', 'back'),
('barbell_row', 'Remo con Barra', 'Barbell Row', 'back'),
('overhead_press', 'Press Militar', 'Overhead Press', 'shoulders'),
('lateral_raise', 'Elevaciones Laterales', 'Lateral Raises', 'shoulders'),
('barbell_curl', 'Curl con Barra', 'Barbell Curl', 'arms'),
('tricep_pushdown', 'Extensión de Tríceps en Polea', 'Tricep Pushdown', 'arms')
ON CONFLICT (id) DO NOTHING;